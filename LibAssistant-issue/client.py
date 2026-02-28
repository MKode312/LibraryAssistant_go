import os
import sys

import grpc

BASE_DIR = os.path.dirname(__file__)
sys.path.insert(0, BASE_DIR)
sys.path.insert(0, os.path.join(BASE_DIR, 'protos'))

import protos.issue_pb2 as issue_pb2
import protos.issue_pb2_grpc as issue_pb2_grpc


def run_examples(address='localhost:50051'):
    channel = grpc.insecure_channel(address)
    stub = issue_pb2_grpc.IssueServiceStub(channel)

    ca = stub.CheckAvailability(
        issue_pb2.CheckAvailabilityRequest(book_id='00000000-0000-0000-0000-000000000001')
    )
    print('availability:', ca.available_copies, ca.total_copies)

    issue = stub.IssueBook(
        issue_pb2.IssueRequest(
            book_id='00000000-0000-0000-0000-000000000001',
            student_id='00000000-0000-0000-0000-000000000002',
            days_due=7,
        )
    )
    print('issue_id:', issue.issue_id)

    debtors = stub.ViewDebtors(issue_pb2.ViewDebtorsRequest(limit=10))
    print('from_cache:', debtors.from_cache)
    for debt in debtors.debts:
        print(debt.issue_id, debt.student_id, debt.book_id, debt.fine)


if __name__ == '__main__':
    run_examples()
