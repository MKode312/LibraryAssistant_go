from concurrent import futures
import grpc
import logging
import sqlite3
from students_db import init_db
from protos import students_pb2_grpc
from protos import students_pb2
import students_db_methods as sdm

ALLOWED_CHARS = 'йцукенгшщзхъфывапролджэячсмитьбю '

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)


class StudentServicer(students_pb2_grpc.StudentsServiceServicer):
    def CreateStudent(self, request, context):
        if not request.full_name:
            logger.error("Student's full name must not be empty")
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("Student's full name must not be empty")
            return students_pb2.CreateStudentResponse()
        if not all(char.lower() in ALLOWED_CHARS for char in request.full_name):
            logger.error("Student's full name must contain only Russian letters")
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("Student's full name must contain only Russian letters")
            return students_pb2.CreateStudentResponse()
        if len(request.full_name.split()) < 2 or len(request.full_name.split()) > 3:
            logger.error("Student's full name must not contain less than 2 words or more than 3 words")
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("Student's full name must not contain less than 2 words or more than 3 words")
            return students_pb2.CreateStudentResponse()
        if request.grade < 1 or request.grade > 11:
            logger.error("Student's grade must be between 1 and 11")
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("Student's grade must be between 1 and 11")
            return students_pb2.CreateStudentResponse()
        if not request.letter:
            logger.error("Student's grade letter must not be empty")
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("Student's grade letter must not be empty")
            return students_pb2.CreateStudentResponse()
        if not all(char.lower() in ALLOWED_CHARS for char in request.letter):
            logger.error("Student's grade letter must contain only Russian letters")
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("Student's grade letter must contain only Russian letters")
            return students_pb2.CreateStudentResponse()
        if len(request.letter.split()) > 1:
            logger.error("Student's grade letter must not contain more that 1 word")
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("Student's grade letter must not contain more that 1 word")
            return students_pb2.CreateStudentResponse()
        try:
            student_id = sdm.CreateStudentIntoDB(request.student_id, request.full_name, request.grade, request.letter)
            logger.info("Student was added")
            return students_pb2.CreateStudentResponse(student_id=student_id)
        except sqlite3.IntegrityError:
            logger.error("Student exists")
            context.set_code(grpc.StatusCode.ALREADY_EXISTS)
            context.set_details("Student exists")
            return students_pb2.CreateStudentResponse()
            
    def DeleteStudentByID(self, request, context):
        status = sdm.DeleteStudentByIDFromDB(request.student_id)
        if not status:
            logger.error("Student not found")
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details("Student not found")
            return students_pb2.DeleteStudentByIDResponse(success=False)
        logger.info("Student was deleted")
        return students_pb2.DeleteStudentByIDResponse(success=True)
    
    def UpdateFullNameOfStudent(self, request, context):
        try:
            if not request.full_name:
                logger.error("Student's full name must not be empty")
                context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
                context.set_details("Student's full name must not be empty")
                return students_pb2.UpdateFullNameOfStudentResponse()
            if not all(char.lower() in ALLOWED_CHARS for char in request.full_name):
                logger.error("Student's full name must contain only Russian letters")
                context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
                context.set_details("Student's full name must contain only Russian letters")
                return students_pb2.UpdateFullNameOfStudentResponse()
            if len(request.full_name.split()) < 2 or len(request.full_name.split()) > 3:
                logger.error("Student's full name must not contain less than 2 words or more than 3 words")
                context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
                context.set_details("Student's full name must not contain less than 2 words or more than 3 words")
                return students_pb2.UpdateFullNameOfStudentResponse()
            student = sdm.UpdateFullNameOfStudentFromDB(request.student_id, request.full_name)
            if student:
                logger.info("Student was changed")
                return students_pb2.UpdateFullNameOfStudentResponse(student=student)
            else:
                logger.error("Student not found")
                context.set_code(grpc.StatusCode.NOT_FOUND)
                context.set_details("Student not found")
                return students_pb2.UpdateFullNameOfStudentResponse()
        except sqlite3.IntegrityError:
            logger.error("Student's full name must be uqnique")
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("Student's full name must be uqnique")
            return students_pb2.UpdateFullNameOfStudentResponse()
    
    def UpdateGradeOfStudent(self, request, context):
        if request.grade < 1 or request.grade > 11:
            logger.error("Student's grade must be between 1 and 11")
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("Student's grade must be between 1 and 11")
            return students_pb2.UpdateGradeOfStudentResponse()
        student = sdm.UpdateGradeOfStudentFromDB(request.student_id, request.grade)
        if not student:
            logger.error("Student not found")
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details("Student not found")
            return students_pb2.UpdateGradeOfStudentResponse()
        logger.info("Student's grade was changed")
        return students_pb2.UpdateGradeOfStudentResponse(student=student)
    
    def UpdateLetterOfStudent(self, request, context):
        if not request.letter:
            logger.error("Student's grade letter must not be empty")
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("Student's grade letter must not be empty")
            return students_pb2.UpdateLetterOfStudentResponse()
        if not all(char.lower() in ALLOWED_CHARS for char in request.letter):
            logger.error("Student's grade letter must contain only Russian letters")
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("Student's grade letter must contain only Russian letters")
            return students_pb2.UpdateLetterOfStudentResponse()
        if len(request.letter.split()) > 1:
            logger.error("Student's grade letter must not contain more that 1 word")
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("Student's grade letter must not contain more that 1 word")
            return students_pb2.UpdateLetterOfStudentResponse()
        student = sdm.UpdateLetterOfStudentFromDB(request.student_id, request.letter)
        if not student:
            logger.error("Student not found")
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details("Student not found")
            return students_pb2.UpdateLetterOfStudentResponse()
        logger.info("Student's grade letter was changed")
        return students_pb2.UpdateLetterOfStudentResponse(student=student)
    
    def GetStudentByID(self, request, context):
        student = sdm.GetStudentByIDFromDB(request.student_id)
        if not student:
            logger.error("Student not found")
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details("Student not found")
            return students_pb2.GetStudentByIDResponse()
        logger.info("Student was received")
        return students_pb2.GetStudentByIDResponse(student=student)
    
    def GetStudentByFullName(self, request, context):
        student = sdm.GetStudentByFullNameFromDB(request.full_name, request.grade, request.letter)
        if not student:
            logger.error("Student not found")
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details("Student not found")
            return students_pb2.GetStudentByFullNameResponse()
        logger.info("Student was received")
        return students_pb2.GetStudentByFullNameResponse(student=student)

    def GetGrade(self, request, context):
        students = sdm.GetGradeFromDB(request.grade, request.letter)
        if not students:
            logger.error("Student not found")
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details("Students not found")
            return students_pb2.GetGradeResponse()
        logger.info("Students was received")
        return students_pb2.GetGradeResponse(student=students)
        
    def GetParallel(self, request, context):
        students = sdm.GetParallelFromDB(request.grade)
        if not students:
            logger.error("Students not found")
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details("Students not found")
            return students_pb2.GetParallelResponse()
        logger.info("Students was received")
        return students_pb2.GetParallelResponse(student=students)
    
    def GetStudentStats(self, request, context):
        return super().GetStudentStats(request, context)

def serve(port=50052):
    init_db()
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    students_pb2_grpc.add_StudentsServiceServicer_to_server(StudentServicer(), server)
    server.add_insecure_port(f'[::]:{port}')
    logger.info(f'Starting Students-Service gRPC on port {port}...')
    server.start()
    logger.info(f'Students-Service gRPC server is running on port {port}')
    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        logger.info('Shutting down Students-Service gRPC server...')
        server.stop(0)
        logger.info('Students-Service gRPC server was stopped')


if __name__ == '__main__':
    serve()