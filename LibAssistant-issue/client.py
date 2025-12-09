import os
import sys
BASE_DIR = os.path.dirname(__file__)
sys.path.insert(0, BASE_DIR)
sys.path.insert(0, os.path.join(BASE_DIR, 'protos'))

import grpc
from google.protobuf import timestamp_pb2
import protos.issue_pb2 as issue_pb2
import protos.issue_pb2_grpc as issue_pb2_grpc


def run_examples(address='localhost:50051'):
    channel = grpc.insecure_channel(address)
    stub = issue_pb2_grpc.IssueServiceStub(channel)

    print('Check availability of book 1')
    ca = stub.CheckAvailability(issue_pb2.CheckAvailabilityRequest(book_id=1))
    print('Available:', ca.available_copies, 'Total:', ca.total_copies)

    print('Issue book 1 to student 1')
    ir = stub.IssueBook(issue_pb2.IssueRequest(book_id=1, student_id=1, days_due=7))
    print('Issue response:', ir.success, ir.message, ir.issue_id)

    print('View debtors')
    vd = stub.ViewDebtors(issue_pb2.ViewDebtorsRequest(limit=10))
    print('From cache:', vd.from_cache)
    for d in vd.debts:
        print(d.issue_id, d.student_name, d.book_title, 'fine=', d.fine, 'overdue=', d.overdue_days)


if __name__ == '__main__':
    run_examples()
