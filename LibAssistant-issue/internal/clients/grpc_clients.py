import logging
import os
import sys
from typing import Any, Dict, Optional

import grpc
import grpc_tools
from grpc_tools import protoc

from internal.config.settings import (
    BOOKS_HOST,
    BOOKS_PORT,
    STUDENTS_HOST,
    STUDENTS_PORT,
)

logger = logging.getLogger(__name__)
PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..'))
PROTO_DIR = os.path.join(PROJECT_ROOT, 'protos')


def _compile_proto(proto_file: str) -> bool:
    if not os.path.exists(proto_file):
        return False
    proto_include = os.path.join(os.path.dirname(grpc_tools.__file__), '_proto')
    result = protoc.main((
        '',
        f'-I{PROTO_DIR}',
        f'-I{proto_include}',
        f'--python_out={PROTO_DIR}',
        f'--grpc_python_out={PROTO_DIR}',
        proto_file,
    ))
    return result == 0


def _ensure_proto(proto_name: str) -> bool:
    proto_file = os.path.join(PROTO_DIR, f'{proto_name}.proto')
    py_out = os.path.join(PROTO_DIR, f'{proto_name}_pb2.py')
    if os.path.exists(py_out):
        return True
    if not os.path.exists(proto_file):
        logger.warning('%s.proto not found in %s', proto_name, PROTO_DIR)
        return False
    if _compile_proto(proto_file):
        return True
    logger.warning('Failed to compile %s.proto', proto_name)
    return False


class StudentsGRPCClient:
    def __init__(self, host: str = STUDENTS_HOST, port: int = STUDENTS_PORT):
        self.host = host
        self.port = port
        self.channel = None
        self.stub = None
        self._is_available = False
        self.last_rpc_code = None
        self._connect()

    def _connect(self):
        if not _ensure_proto('students'):
            return

        try:
            sys.path.insert(0, PROJECT_ROOT)
            sys.path.insert(0, PROTO_DIR)
            import protos.students_pb2_grpc as students_pb2_grpc  # type: ignore

            if self.host not in ('localhost', '127.0.0.1'):
                self.channel = grpc.insecure_channel(f'{self.host}:{self.port}')
            else:
                self.channel = grpc.insecure_channel(f'{self.host}:{self.port}')

            self.stub = students_pb2_grpc.StudentsServiceStub(self.channel)
            self._is_available = True
            logger.info('Connected to Students service at %s:%s', self.host, self.port)
        except Exception as exc:
            logger.warning('Failed to connect to Students service: %s', exc)
            self._is_available = False

    def is_available(self) -> bool:
        return self._is_available

    def get_student_by_id(self, student_id: str) -> Optional[Dict[str, Any]]:
        if not self._is_available:
            self.last_rpc_code = grpc.StatusCode.UNAVAILABLE
            return None
        try:
            import protos.students_pb2 as students_pb2  # type: ignore

            response = self.stub.GetStudentByID(
                students_pb2.GetStudentByIDRequest(student_id=student_id),
                timeout=5,
            )
            student = response.student
            if not student or not student.id:
                self.last_rpc_code = grpc.StatusCode.NOT_FOUND
                return None
            self.last_rpc_code = None
            return {
                'id': student.id,
                'name': student.full_name,
                'grade': student.grade,
                'letter': student.letter,
            }
        except grpc.RpcError as exc:
            self.last_rpc_code = exc.code()
            logger.warning('Students gRPC error: %s %s', exc.code(), exc.details())
            return None
        except Exception as exc:
            self.last_rpc_code = grpc.StatusCode.UNKNOWN
            logger.error('Error getting student %s: %s', student_id, exc, exc_info=True)
            return None

    def close(self):
        if self.channel:
            self.channel.close()


class BooksGRPCClient:
    def __init__(self, host: str = BOOKS_HOST, port: int = BOOKS_PORT):
        self.host = host
        self.port = port
        self.channel = None
        self.stub = None
        self._is_available = False
        self.last_rpc_code = None
        self._connect()

    def _connect(self):
        if not _ensure_proto('books'):
            return

        try:
            sys.path.insert(0, PROJECT_ROOT)
            sys.path.insert(0, PROTO_DIR)
            import protos.books_pb2_grpc as books_pb2_grpc  # type: ignore

            if self.host not in ('localhost', '127.0.0.1'):
                self.channel = grpc.insecure_channel(f'{self.host}:{self.port}')
            else:
                self.channel = grpc.insecure_channel(f'{self.host}:{self.port}')

            self.stub = books_pb2_grpc.BooksStub(self.channel)
            self._is_available = True
            logger.info('Connected to Books service at %s:%s', self.host, self.port)
        except Exception as exc:
            logger.warning('Failed to connect to Books service: %s', exc)
            self._is_available = False

    def is_available(self) -> bool:
        return self._is_available

    def get_book_by_id(self, book_id: str) -> Optional[Dict[str, Any]]:
        if not self._is_available:
            self.last_rpc_code = grpc.StatusCode.UNAVAILABLE
            return None
        try:
            import protos.books_pb2 as books_pb2  # type: ignore

            response = self.stub.GetBookByID(books_pb2.GetBookByIDRequest(book_id=book_id), timeout=5)
            book = response.book
            book_id_value = getattr(book, 'ID', '')
            if not book_id_value:
                self.last_rpc_code = grpc.StatusCode.NOT_FOUND
                return None
            self.last_rpc_code = None
            return {
                'id': book_id_value,
                'title': book.title,
                'genre': book.genre,
                'available_copies': book.available_copies,
                'total_copies': book.available_copies,
            }
        except grpc.RpcError as exc:
            self.last_rpc_code = exc.code()
            logger.warning('Books gRPC error: %s %s', exc.code(), exc.details())
            return None
        except Exception as exc:
            self.last_rpc_code = grpc.StatusCode.UNKNOWN
            logger.error('Error getting book %s: %s', book_id, exc, exc_info=True)
            return None

    def check_availability(self, book_id: str) -> Optional[Dict[str, int]]:
        book = self.get_book_by_id(book_id)
        if not book:
            return None
        return {
            'available_copies': book.get('available_copies', 0),
            'total_copies': book.get('total_copies', 0),
        }

    def take_book(self, book_id: str, copies: int = 1) -> bool:
        if not self._is_available:
            self.last_rpc_code = grpc.StatusCode.UNAVAILABLE
            return False
        try:
            import protos.books_pb2 as books_pb2  # type: ignore

            response = self.stub.TakeBook(
                books_pb2.TakeBookRequest(book_id=book_id, take_copies=copies),
                timeout=5,
            )
            self.last_rpc_code = None
            return response.success
        except grpc.RpcError as exc:
            self.last_rpc_code = exc.code()
            logger.warning('Books gRPC error: %s %s', exc.code(), exc.details())
            return False
        except Exception as exc:
            self.last_rpc_code = grpc.StatusCode.UNKNOWN
            logger.error('Error taking book %s: %s', book_id, exc, exc_info=True)
            return False

    def add_book(self, title: str, copies: int, genre: str = 'unknown') -> Optional[str]:
        if not self._is_available:
            self.last_rpc_code = grpc.StatusCode.UNAVAILABLE
            return None
        try:
            import protos.books_pb2 as books_pb2  # type: ignore

            response = self.stub.AddBook(
                books_pb2.AddBookRequest(genre=genre, title=title, quantity=copies),
                timeout=5,
            )
            self.last_rpc_code = None
            return response.book_id or None
        except grpc.RpcError as exc:
            self.last_rpc_code = exc.code()
            logger.warning('Books gRPC error: %s %s', exc.code(), exc.details())
            return None
        except Exception as exc:
            self.last_rpc_code = grpc.StatusCode.UNKNOWN
            logger.error('Error adding book %s: %s', title, exc, exc_info=True)
            return None

    def return_book(self, book_id: str, copies: int = 1) -> bool:
        if not self._is_available:
            self.last_rpc_code = grpc.StatusCode.UNAVAILABLE
            return False
        try:
            import protos.books_pb2 as books_pb2  # type: ignore

            response = self.stub.AddCopies(
                books_pb2.AddCopiesRequest(book_id=book_id, copies_to_add=copies),
                timeout=5,
            )
            self.last_rpc_code = None
            return response.success
        except grpc.RpcError as exc:
            self.last_rpc_code = exc.code()
            logger.warning('Books gRPC error: %s %s', exc.code(), exc.details())
            return False
        except Exception as exc:
            self.last_rpc_code = grpc.StatusCode.UNKNOWN
            logger.error('Error returning book %s: %s', book_id, exc, exc_info=True)
            return False

    def close(self):
        if self.channel:
            self.channel.close()


_students_client: Optional[StudentsGRPCClient] = None
_books_client: Optional[BooksGRPCClient] = None


def get_students_client() -> StudentsGRPCClient:
    global _students_client
    if _students_client is None:
        _students_client = StudentsGRPCClient()
    return _students_client


def get_books_client() -> BooksGRPCClient:
    global _books_client
    if _books_client is None:
        _books_client = BooksGRPCClient()
    return _books_client


def close_all_clients():
    global _students_client, _books_client
    if _students_client:
        _students_client.close()
        _students_client = None
    if _books_client:
        _books_client.close()
        _books_client = None
