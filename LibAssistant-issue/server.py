import os
import time
import logging
from concurrent import futures
from datetime import datetime, timedelta
import importlib
import sys

from grpc_tools import protoc

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('logs/issue_service.log'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

BASE_DIR = os.path.dirname(__file__)
PROTO_DIR = os.path.join(BASE_DIR, 'protos')
LOGS_DIR = os.path.join(BASE_DIR, 'logs')

# Create logs directory if it doesn't exist
os.makedirs(LOGS_DIR, exist_ok=True)


def compile_proto():
    # generate pb2 and pb2_grpc under protos/
    proto_file = os.path.join(PROTO_DIR, 'issue.proto')
    out_py = PROTO_DIR
    if not os.path.exists(os.path.join(PROTO_DIR, 'issue_pb2.py')):
        logger.info('Compiling proto...')
        res = protoc.main((
            '',
            f'-I{PROTO_DIR}',
            f'--python_out={PROTO_DIR}',
            f'--grpc_python_out={PROTO_DIR}',
            proto_file,
        ))
        if res != 0:
            logger.error('protoc failed')
            raise RuntimeError('protoc failed')
        logger.info('Proto compiled successfully')


compile_proto()

sys.path.insert(0, BASE_DIR)
sys.path.insert(0, PROTO_DIR)
import protos.issue_pb2 as issue_pb2
import protos.issue_pb2_grpc as issue_pb2_grpc
from google.protobuf import timestamp_pb2

from db import SessionLocal, init_db, Book, Student, Issue, update_all_overdues
from cache import load_debtors_cache, save_debtors_cache

import grpc


class IssueServicer(issue_pb2_grpc.IssueServiceServicer):
    def __init__(self):
        init_db(seed=True)

    def IssueBook(self, request, context):
        logger.info(f'IssueBook request: book_id={request.book_id}, student_id={request.student_id}, days_due={request.days_due}')
        session = SessionLocal()
        try:
            # transaction
            book = session.query(Book).filter(Book.id == request.book_id).with_for_update().one_or_none()
            if not book:
                logger.warning(f'Book not found: book_id={request.book_id}')
                return issue_pb2.IssueResponse(success=False, message='Book not found')
            if book.available_copies <= 0:
                logger.warning(f'No available copies: book_id={request.book_id}')
                return issue_pb2.IssueResponse(success=False, message='No available copies')
            student = session.query(Student).filter(Student.id == request.student_id).one_or_none()
            if not student:
                logger.warning(f'Student not found: student_id={request.student_id}')
                return issue_pb2.IssueResponse(success=False, message='Student not found')
            # decrement and create issue
            book.available_copies -= 1
            days = request.days_due if request.days_due > 0 else 14
            issue = Issue(book_id=book.id, student_id=student.id,
                          issue_date=datetime.utcnow(), due_date=datetime.utcnow() + timedelta(days=days))
            session.add(issue)
            session.commit()
            logger.info(f'Book issued successfully: issue_id={issue.id}, book_id={book.id}, student_id={student.id}')
            return issue_pb2.IssueResponse(success=True, message='Issued', issue_id=issue.id)
        except Exception as e:
            session.rollback()
            logger.error(f'Error in IssueBook: {e}', exc_info=True)
            return issue_pb2.IssueResponse(success=False, message=f'Error: {e}')
        finally:
            session.close()

    def CheckAvailability(self, request, context):
        session = SessionLocal()
        try:
            book = session.query(Book).filter(Book.id == request.book_id).one_or_none()
            if not book:
                return issue_pb2.CheckAvailabilityResponse(available_copies=0, total_copies=0)
            return issue_pb2.CheckAvailabilityResponse(available_copies=book.available_copies, total_copies=book.total_copies)
        finally:
            session.close()

    def ReturnBook(self, request, context):
        session = SessionLocal()
        try:
            issue = session.query(Issue).filter(Issue.id == request.issue_id).with_for_update().one_or_none()
            if not issue:
                return issue_pb2.ReturnResponse(success=False, message='Issue not found')
            if issue.return_date:
                return issue_pb2.ReturnResponse(success=False, message='Already returned')
            issue.return_date = datetime.utcnow()
            # increment book copy
            book = session.query(Book).filter(Book.id == issue.book_id).one_or_none()
            if book and not issue.lost:
                book.available_copies += 1
            # calc overdue/fine
            update_all_overdues(session)
            fine = issue.fine
            session.commit()
            return issue_pb2.ReturnResponse(success=True, message='Returned', fine=fine)
        except Exception as e:
            session.rollback()
            return issue_pb2.ReturnResponse(success=False, message=f'Error: {e}')
        finally:
            session.close()

    def ReportLost(self, request, context):
        session = SessionLocal()
        try:
            issue = session.query(Issue).filter(Issue.id == request.issue_id).with_for_update().one_or_none()
            if not issue:
                return issue_pb2.ReportLostResponse(success=False, message='Issue not found')
            if issue.lost:
                return issue_pb2.ReportLostResponse(success=False, message='Already reported lost')
            issue.lost = True
            # do not return copy to library
            update_all_overdues(session)
            fine = issue.fine
            session.commit()
            return issue_pb2.ReportLostResponse(success=True, message='Reported lost', fine=fine)
        except Exception as e:
            session.rollback()
            return issue_pb2.ReportLostResponse(success=False, message=f'Error: {e}')
        finally:
            session.close()

    def GetAllDebts(self, request, context):
        session = SessionLocal()
        try:
            update_all_overdues(session)
            issues = session.query(Issue).filter(Issue.return_date == None).all()
            debts = []
            for it in issues:
                debts.append(issue_pb2.Debt(
                    issue_id=it.id,
                    student_id=it.student_id,
                    student_name=it.student.name,
                    book_id=it.book_id,
                    book_title=it.book.title,
                    due_date=timestamp_pb2.Timestamp(seconds=int(it.due_date.timestamp())) if it.due_date else None,
                    overdue_days=it.overdue_days,
                    fine=it.fine
                ))
            return issue_pb2.GetAllDebtsResponse(debts=debts)
        finally:
            session.close()

    def ViewDebtors(self, request, context):
        # try cache
        cached = load_debtors_cache()
        if cached is not None:
            # return cached (limited)
            arr = cached[:request.limit] if request.limit and request.limit > 0 else cached
            debts = []
            for d in arr:
                # convert back
                ts = None
                if d.get('due_date'):
                    try:
                        dt = datetime.fromisoformat(d.get('due_date'))
                        ts = timestamp_pb2.Timestamp(seconds=int(dt.timestamp()))
                    except Exception:
                        ts = None
                debts.append(issue_pb2.Debt(issue_id=d.get('issue_id', 0), student_id=d.get('student_id', 0), student_name=d.get('student_name',''), book_id=d.get('book_id',0), book_title=d.get('book_title',''), due_date=ts, overdue_days=d.get('overdue_days',0), fine=d.get('fine',0)))
            return issue_pb2.ViewDebtorsResponse(debts=debts, from_cache=True)

        session = SessionLocal()
        try:
            update_all_overdues(session)
            issues = session.query(Issue).filter(Issue.return_date == None).order_by(Issue.fine.desc()).all()
            debts = []
            serializable = []
            for it in issues:
                due_iso = it.due_date.isoformat() if it.due_date else None
                serializable.append({'issue_id': it.id, 'student_id': it.student_id, 'student_name': it.student.name, 'book_id': it.book_id, 'book_title': it.book.title, 'due_date': due_iso, 'overdue_days': it.overdue_days, 'fine': it.fine})
                ts = timestamp_pb2.Timestamp(seconds=int(it.due_date.timestamp())) if it.due_date else None
                debts.append(issue_pb2.Debt(issue_id=it.id, student_id=it.student_id, student_name=it.student.name, book_id=it.book_id, book_title=it.book.title, due_date=ts, overdue_days=it.overdue_days, fine=it.fine))
            # save cache
            save_debtors_cache(serializable)
            arr = debts[:request.limit] if request.limit and request.limit > 0 else debts
            return issue_pb2.ViewDebtorsResponse(debts=arr, from_cache=False)
        finally:
            session.close()

    def AddBook(self, request, context):
        logger.info(f'AddBook request: title={request.title}, total_copies={request.total_copies}')
        session = SessionLocal()
        try:
            book = Book(title=request.title, total_copies=request.total_copies, available_copies=request.total_copies)
            session.add(book)
            session.commit()
            logger.info(f'Book added successfully: book_id={book.id}, title={book.title}')
            return issue_pb2.AddBookResponse(success=True, message='Book added successfully', book_id=book.id)
        except Exception as e:
            session.rollback()
            logger.error(f'Error in AddBook: {e}', exc_info=True)
            return issue_pb2.AddBookResponse(success=False, message=f'Error: {e}', book_id=0)
        finally:
            session.close()



def serve(port=50051):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    issue_pb2_grpc.add_IssueServiceServicer_to_server(IssueServicer(), server)
    server.add_insecure_port(f'[::]:{port}')
    logger.info(f'Starting Issue-Service gRPC on port {port}...')
    server.start()
    logger.info(f'Issue-Service gRPC server is running on port {port}')
    try:
        while True:
            time.sleep(60)
    except KeyboardInterrupt:
        logger.info('Shutting down Issue-Service gRPC server...')
        server.stop(0)
        logger.info('Issue-Service gRPC server stopped')


if __name__ == '__main__':
    serve()
