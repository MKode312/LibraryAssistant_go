import json
import logging
import os
import sys
import time
from concurrent import futures
from datetime import datetime, timedelta
from typing import Dict, List

import grpc
import grpc_tools
from google.protobuf import timestamp_pb2
from grpc_tools import protoc
from sqlalchemy.exc import IntegrityError

from internal.clients.grpc_clients import close_all_clients, get_books_client, get_students_client
from internal.config.settings import DEFAULT_ISSUE_DAYS, HOST, PORT
from internal.service.status_codes import ErrorHandler
from internal.storage.cache import load_debtors_cache, save_debtors_cache
from internal.storage.db import Issue, SessionLocal, Student, init_db, update_all_overdues

PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..', '..'))
PROTO_DIR = os.path.join(PROJECT_ROOT, 'protos')
LOGS_DIR = os.path.join(PROJECT_ROOT, 'logs')
os.makedirs(LOGS_DIR, exist_ok=True)

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler(os.path.join(LOGS_DIR, 'issue_service.log')),
        logging.StreamHandler(),
    ],
)
logger = logging.getLogger(__name__)


def _has_id(value: str) -> bool:
    return value is not None and str(value).strip() != ''


def compile_proto():
    proto_files = [os.path.join(PROTO_DIR, f) for f in os.listdir(PROTO_DIR) if f.endswith('.proto')]
    if not proto_files:
        raise RuntimeError('No proto files found')

    need_compile = False
    for proto_file in proto_files:
        base = os.path.splitext(os.path.basename(proto_file))[0]
        if not os.path.exists(os.path.join(PROTO_DIR, f'{base}_pb2.py')):
            need_compile = True
            break

    if not need_compile:
        return

    logger.info('Compiling proto files')
    proto_include = os.path.join(os.path.dirname(grpc_tools.__file__), '_proto')
    result = protoc.main((
        '',
        f'-I{PROTO_DIR}',
        f'-I{proto_include}',
        f'--python_out={PROTO_DIR}',
        f'--grpc_python_out={PROTO_DIR}',
        *proto_files,
    ))
    if result != 0:
        raise RuntimeError('protoc failed')


compile_proto()
sys.path.insert(0, PROJECT_ROOT)
sys.path.insert(0, PROTO_DIR)
import protos.issue_pb2 as issue_pb2
import protos.issue_pb2_grpc as issue_pb2_grpc


class IssueServicer(issue_pb2_grpc.IssueServiceServicer):
    def __init__(self):
        init_db()
        self.students_client = get_students_client()
        self.books_client = get_books_client()

    def IssueBook(self, request, context):
        session = SessionLocal()
        try:
            if not _has_id(request.book_id):
                ErrorHandler.set_custom_error(
                    context,
                    grpc.StatusCode.INVALID_ARGUMENT,
                    'book_id is required',
                )
                return issue_pb2.IssueResponse()
            if not _has_id(request.student_id):
                ErrorHandler.set_custom_error(
                    context,
                    grpc.StatusCode.INVALID_ARGUMENT,
                    'student_id is required',
                )
                return issue_pb2.IssueResponse()

            student_info = self.students_client.get_student_by_id(request.student_id)
            if not student_info:
                logger.warning(
                    'IssueBook student lookup failed: student_id=%s rpc_code=%s',
                    request.student_id,
                    self.students_client.last_rpc_code,
                )
                if self.students_client.last_rpc_code not in (None, grpc.StatusCode.NOT_FOUND):
                    ErrorHandler.set_error(context, 'EXTERNAL_SERVICE_ERROR')
                    return issue_pb2.IssueResponse()
                ErrorHandler.set_error(context, 'STUDENT_NOT_FOUND')
                return issue_pb2.IssueResponse()

            book_info = self.books_client.get_book_by_id(request.book_id)
            if not book_info:
                if self.books_client.last_rpc_code not in (None, grpc.StatusCode.NOT_FOUND):
                    ErrorHandler.set_error(context, 'EXTERNAL_SERVICE_ERROR')
                    return issue_pb2.IssueResponse()
                ErrorHandler.set_error(context, 'BOOK_NOT_FOUND')
                return issue_pb2.IssueResponse()
            if book_info.get('available_copies', 0) <= 0:
                ErrorHandler.set_error(context, 'NO_AVAILABLE_COPIES')
                return issue_pb2.IssueResponse()

            days = request.days_due if request.days_due > 0 else DEFAULT_ISSUE_DAYS
            if days < 1 or days > 365:
                ErrorHandler.set_error(context, 'INVALID_DAYS_DUE')
                return issue_pb2.IssueResponse()

            student = session.query(Student).filter(Student.id == request.student_id).one_or_none()
            if not student:
                student = Student(id=request.student_id, name=student_info.get('name', request.student_id))
                session.add(student)

            active_issue = (
                session.query(Issue)
                .filter(
                    Issue.book_id == request.book_id,
                    Issue.student_id == request.student_id,
                    Issue.return_date == None,
                )
                .with_for_update()
                .one_or_none()
            )
            if active_issue is not None:
                ErrorHandler.set_custom_error(
                    context,
                    grpc.StatusCode.FAILED_PRECONDITION,
                    'This student already has an active issue for this book',
                )
                return issue_pb2.IssueResponse()

            if not self.books_client.take_book(request.book_id, copies=1):
                if self.books_client.last_rpc_code not in (None, grpc.StatusCode.NOT_FOUND):
                    ErrorHandler.set_error(context, 'EXTERNAL_SERVICE_ERROR')
                    return issue_pb2.IssueResponse()
                ErrorHandler.set_error(context, 'EXTERNAL_SERVICE_ERROR')
                return issue_pb2.IssueResponse()

            now = datetime.utcnow()
            issue = Issue(
                id=str(uuid.uuid4()),
                book_id=request.book_id,
                student_id=student.id,
                issue_date=now,
                due_date=now + timedelta(days=days),
            )
            session.add(issue)
            session.commit()

            self._emit_event('book.issued', {
                'issue_id': issue.id,
                'book_id': request.book_id,
                'student_id': student.id,
                'issue_date': now.isoformat(),
                'due_date': issue.due_date.isoformat(),
            })
            return issue_pb2.IssueResponse(issue_id=issue.id)
        except IntegrityError as exc:
            session.rollback()
            if 'uq_active_issue_student_book' in str(exc):
                ErrorHandler.set_custom_error(
                    context,
                    grpc.StatusCode.FAILED_PRECONDITION,
                    'This student already has an active issue for this book',
                )
                return issue_pb2.IssueResponse()
            logger.error('IssueBook integrity error: %s', exc, exc_info=True)
            ErrorHandler.set_error(context, 'DATABASE_ERROR')
            return issue_pb2.IssueResponse()
        except Exception as exc:
            session.rollback()
            logger.error('IssueBook failed: %s', exc, exc_info=True)
            ErrorHandler.set_error(context, 'DATABASE_ERROR')
            return issue_pb2.IssueResponse()
        finally:
            session.close()

    def CheckAvailability(self, request, context):
        try:
            availability = self.books_client.check_availability(request.book_id)
            if not availability:
                if self.books_client.last_rpc_code not in (None, grpc.StatusCode.NOT_FOUND):
                    ErrorHandler.set_error(context, 'EXTERNAL_SERVICE_ERROR')
                    return issue_pb2.CheckAvailabilityResponse()
                ErrorHandler.set_error(context, 'BOOK_NOT_FOUND')
                return issue_pb2.CheckAvailabilityResponse()
            return issue_pb2.CheckAvailabilityResponse(
                available_copies=availability.get('available_copies', 0),
                total_copies=availability.get('total_copies', 0),
            )
        except Exception as exc:
            logger.error('CheckAvailability failed: %s', exc, exc_info=True)
            ErrorHandler.set_error(context, 'DATABASE_ERROR')
            return issue_pb2.CheckAvailabilityResponse()

    def ReturnBook(self, request, context):
        session = SessionLocal()
        try:
            issue = session.query(Issue).filter(Issue.id == request.issue_id).with_for_update().one_or_none()
            if not issue:
                ErrorHandler.set_error(context, 'ISSUE_NOT_FOUND')
                return issue_pb2.ReturnResponse()
            if issue.return_date:
                ErrorHandler.set_error(context, 'ALREADY_RETURNED')
                return issue_pb2.ReturnResponse()

            issue.return_date = datetime.utcnow()
            if not issue.lost:
                if not self.books_client.return_book(issue.book_id, copies=1):
                    ErrorHandler.set_error(context, 'EXTERNAL_SERVICE_ERROR')
                    return issue_pb2.ReturnResponse()

            update_all_overdues(session)
            fine = issue.fine
            session.commit()

            self._emit_event('book.returned', {
                'issue_id': issue.id,
                'book_id': issue.book_id,
                'student_id': issue.student_id,
                'return_date': issue.return_date.isoformat(),
                'fine': fine,
            })
            return issue_pb2.ReturnResponse(fine=fine)
        except Exception as exc:
            session.rollback()
            logger.error('ReturnBook failed: %s', exc, exc_info=True)
            ErrorHandler.set_error(context, 'DATABASE_ERROR')
            return issue_pb2.ReturnResponse()
        finally:
            session.close()

    def ReportLost(self, request, context):
        session = SessionLocal()
        try:
            issue = session.query(Issue).filter(Issue.id == request.issue_id).with_for_update().one_or_none()
            if not issue:
                ErrorHandler.set_error(context, 'ISSUE_NOT_FOUND')
                return issue_pb2.ReportLostResponse()
            if issue.lost:
                ErrorHandler.set_error(context, 'ALREADY_REPORTED_LOST')
                return issue_pb2.ReportLostResponse()
            if issue.return_date:
                ErrorHandler.set_error(context, 'ALREADY_RETURNED')
                return issue_pb2.ReportLostResponse()

            issue.lost = True
            issue.return_date = datetime.utcnow()
            update_all_overdues(session)
            fine = issue.fine
            session.commit()
            return issue_pb2.ReportLostResponse(fine=fine)
        except Exception as exc:
            session.rollback()
            logger.error('ReportLost failed: %s', exc, exc_info=True)
            ErrorHandler.set_error(context, 'DATABASE_ERROR')
            return issue_pb2.ReportLostResponse()
        finally:
            session.close()

    def GetAllDebts(self, request, context):
        session = SessionLocal()
        try:
            update_all_overdues(session)
            issues = session.query(Issue).filter(Issue.return_date == None).all()
            title_cache: Dict[str, str] = {}
            debts = []
            for item in issues:
                book_title = self._resolve_book_title(item.book_id, title_cache)
                ts = timestamp_pb2.Timestamp(seconds=int(item.due_date.timestamp())) if item.due_date else None
                debt = issue_pb2.Debt(
                    issue_id=item.id,
                    student_id=item.student_id,
                    student_name=item.student.name if item.student else item.student_id,
                    book_id=item.book_id,
                    book_title=book_title,
                    overdue_days=item.overdue_days,
                    fine=item.fine,
                )
                if ts is not None:
                    debt.due_date.CopyFrom(ts)
                debts.append(debt)
            return issue_pb2.GetAllDebtsResponse(debts=debts)
        except Exception as exc:
            logger.error('GetAllDebts failed: %s', exc, exc_info=True)
            ErrorHandler.set_error(context, 'DATABASE_ERROR')
            return issue_pb2.GetAllDebtsResponse(debts=[])
        finally:
            session.close()

    def ViewDebtors(self, request, context):
        cached = load_debtors_cache()
        if cached is not None:
            debts = self._build_debts_from_cache(cached, request.limit)
            return issue_pb2.ViewDebtorsResponse(debts=debts, from_cache=True)

        session = SessionLocal()
        try:
            update_all_overdues(session)
            issues = session.query(Issue).filter(Issue.return_date == None).order_by(Issue.fine.desc()).all()
            title_cache: Dict[str, str] = {}

            debts = []
            serializable = []
            for item in issues:
                book_title = self._resolve_book_title(item.book_id, title_cache)
                serializable.append({
                    'issue_id': item.id,
                    'student_id': item.student_id,
                    'student_name': item.student.name if item.student else item.student_id,
                    'book_id': item.book_id,
                    'book_title': book_title,
                    'due_date': item.due_date.isoformat() if item.due_date else None,
                    'overdue_days': item.overdue_days,
                    'fine': item.fine,
                })

                ts = timestamp_pb2.Timestamp(seconds=int(item.due_date.timestamp())) if item.due_date else None
                debt = issue_pb2.Debt(
                    issue_id=item.id,
                    student_id=item.student_id,
                    student_name=item.student.name if item.student else item.student_id,
                    book_id=item.book_id,
                    book_title=book_title,
                    overdue_days=item.overdue_days,
                    fine=item.fine,
                )
                if ts is not None:
                    debt.due_date.CopyFrom(ts)
                debts.append(debt)

            save_debtors_cache(serializable)
            data = debts[:request.limit] if request.limit and request.limit > 0 else debts
            return issue_pb2.ViewDebtorsResponse(debts=data, from_cache=False)
        except Exception as exc:
            logger.error('ViewDebtors failed: %s', exc, exc_info=True)
            ErrorHandler.set_error(context, 'DATABASE_ERROR')
            return issue_pb2.ViewDebtorsResponse(debts=[], from_cache=False)
        finally:
            session.close()

    def AddBook(self, request, context):
        try:
            if not request.title or not request.title.strip():
                ErrorHandler.set_custom_error(context, grpc.StatusCode.INVALID_ARGUMENT, 'Invalid book title')
                return issue_pb2.AddBookResponse()
            if request.total_copies <= 0:
                ErrorHandler.set_custom_error(context, grpc.StatusCode.INVALID_ARGUMENT, 'Invalid number of copies')
                return issue_pb2.AddBookResponse()

            book_id = self.books_client.add_book(request.title, request.total_copies, genre='unknown')
            if not book_id:
                ErrorHandler.set_error(context, 'EXTERNAL_SERVICE_ERROR')
                return issue_pb2.AddBookResponse()
            return issue_pb2.AddBookResponse(book_id=book_id)
        except Exception as exc:
            logger.error('AddBook failed: %s', exc, exc_info=True)
            ErrorHandler.set_error(context, 'DATABASE_ERROR')
            return issue_pb2.AddBookResponse()

    @staticmethod
    def _build_debts_from_cache(cached: List[Dict], limit: int) -> List:
        debts = []
        for item in cached:
            debt = issue_pb2.Debt(
                issue_id=item.get('issue_id', ''),
                student_id=item.get('student_id', ''),
                student_name=item.get('student_name', ''),
                book_id=item.get('book_id', ''),
                book_title=item.get('book_title', ''),
                overdue_days=item.get('overdue_days', 0),
                fine=item.get('fine', 0),
            )
            if item.get('due_date'):
                try:
                    dt = datetime.fromisoformat(item['due_date'])
                    debt.due_date.CopyFrom(timestamp_pb2.Timestamp(seconds=int(dt.timestamp())))
                except Exception:
                    pass
            debts.append(debt)
        return debts[:limit] if limit and limit > 0 else debts

    def _resolve_book_title(self, book_id: str, title_cache: Dict[str, str]) -> str:
        if book_id in title_cache:
            return title_cache[book_id]
        book_info = self.books_client.get_book_by_id(book_id)
        title = book_info.get('title', book_id) if book_info else book_id
        title_cache[book_id] = title
        return title

    @staticmethod
    def _emit_event(event_type: str, data: Dict):
        try:
            logger.info('Event emitted: %s - %s', event_type, json.dumps(data, ensure_ascii=False))
        except Exception as exc:
            logger.warning('Failed to emit event: %s', exc)


def serve(host: str = HOST, port: int = PORT):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    issue_pb2_grpc.add_IssueServiceServicer_to_server(IssueServicer(), server)
    server.add_insecure_port(f'{host}:{port}')
    server.start()
    logger.info('Issue service gRPC started on %s:%s', host, port)
    try:
        while True:
            time.sleep(60)
    except KeyboardInterrupt:
        close_all_clients()


if __name__ == '__main__':
    serve()
