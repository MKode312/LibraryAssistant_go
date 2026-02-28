import grpc

GRPC_STATUS_MAPPING = {
    'BOOK_NOT_FOUND': grpc.StatusCode.NOT_FOUND,
    'STUDENT_NOT_FOUND': grpc.StatusCode.NOT_FOUND,
    'NO_AVAILABLE_COPIES': grpc.StatusCode.RESOURCE_EXHAUSTED,
    'INVALID_DAYS_DUE': grpc.StatusCode.INVALID_ARGUMENT,
    'DATABASE_ERROR': grpc.StatusCode.INTERNAL,
    'EXTERNAL_SERVICE_ERROR': grpc.StatusCode.UNAVAILABLE,
    'ISSUE_NOT_FOUND': grpc.StatusCode.NOT_FOUND,
    'BOOK_ALREADY_RETURNED': grpc.StatusCode.FAILED_PRECONDITION,
    'ALREADY_REPORTED_LOST': grpc.StatusCode.FAILED_PRECONDITION,
    'ALREADY_RETURNED': grpc.StatusCode.FAILED_PRECONDITION,
    'NO_DEBTS': grpc.StatusCode.OK,
    'CACHE_ERROR': grpc.StatusCode.INTERNAL,
}

ERROR_MESSAGES = {
    'BOOK_NOT_FOUND': 'Book not found',
    'STUDENT_NOT_FOUND': 'Student not found',
    'NO_AVAILABLE_COPIES': 'No available copies of this book',
    'INVALID_DAYS_DUE': 'Invalid number of days (must be between 1 and 365)',
    'DATABASE_ERROR': 'Database error',
    'EXTERNAL_SERVICE_ERROR': 'External service error',
    'UNKNOWN_ERROR': 'Unknown error',
    'ISSUE_NOT_FOUND': 'Issue record not found',
    'BOOK_ALREADY_RETURNED': 'Book has already been returned',
    'ALREADY_REPORTED_LOST': 'Book already reported as lost',
    'ALREADY_RETURNED': 'Book has been returned, cannot report as lost',
    'NO_DEBTS': 'No debts found',
    'CACHE_ERROR': 'Cache error',
}


class ErrorHandler:
    @staticmethod
    def set_error(context, error_key: str):
        status_code = GRPC_STATUS_MAPPING.get(error_key, grpc.StatusCode.INTERNAL)
        message = ERROR_MESSAGES.get(error_key, 'Unknown error')
        context.set_code(status_code)
        context.set_details(message)

    @staticmethod
    def set_custom_error(context, status_code: grpc.StatusCode, message: str):
        context.set_code(status_code)
        context.set_details(message)
