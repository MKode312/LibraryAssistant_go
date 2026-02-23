from students_db import get_connection
from protos import students_pb2


def CreateStudentIntoDB(id: str, full_name: str, grade: int, letter: str) -> str:
    with get_connection() as con:
        cur = con.cursor()
        cur.execute(
            "INSERT INTO students (id, full_name, grade, letter) VALUES (?, ?, ?, ?)",
            (id, full_name, grade, letter))
        con.commit()
        return id

def DeleteStudentByIDFromDB(id: str) -> bool:
    with get_connection() as con:
        cur = con.cursor()
        cur.execute(
            "DELETE FROM students WHERE id=?",
            (id,))
        if cur.rowcount:
            con.commit()
            return True
        return False

def UpdateFullNameOfStudentFromDB(id: str, full_name: str) -> students_pb2.Student:
    with get_connection() as con:
        cur = con.cursor()
        cur.execute(
            "UPDATE students SET full_name=? WHERE id=?",
            (full_name, id))
        if cur.rowcount:
            con.commit()
            return GetStudentByIDFromDB(id)
        return None

def UpdateGradeOfStudentFromDB(id: str, grade: int) -> students_pb2.Student:
    with get_connection() as con:
        cur = con.cursor()
        cur.execute(
            "UPDATE students SET grade=? WHERE id=?",
            (grade, id))
        if cur.rowcount:
            con.commit()
            return GetStudentByIDFromDB(id)
        return None

def UpdateLetterOfStudentFromDB(id: str, letter: str) -> students_pb2.Student:
    with get_connection() as con:
        cur = con.cursor()
        cur.execute(
            "UPDATE students SET letter=? WHERE id=?",
            (letter, id))
        if cur.rowcount:
            con.commit()
            return GetStudentByIDFromDB(id)
        return None

def GetStudentByIDFromDB(id: str) -> students_pb2.Student:
    with get_connection() as con:
        cur = con.cursor()
        cur.execute(
            "SELECT id, full_name, grade, letter FROM students WHERE id=?",
            (id,))
        row = cur.fetchone()
        if row:
            student = students_pb2.Student(
                id=row[0],
                full_name=row[1],
                grade=row[2],
                letter=row[3])
            return student
        return None

def GetStudentByFullNameFromDB(full_name: str, grade: int, letter:str) -> students_pb2.Student:
    with get_connection() as con:
        cur = con.cursor()
        cur.execute(
            "SELECT id, full_name, grade, letter FROM students WHERE full_name=? AND grade=? AND letter=?",
            (full_name, grade, letter))
        rows = cur.fetchone()
        if rows:
            student = students_pb2.Student(
                id=rows[0],
                full_name=rows[1],
                grade=rows[2],
                letter=rows[3])
            return student
        return None
    
def GetGradeFromDB(grade: int, letter: str) -> list[students_pb2.Student]:
    with get_connection() as con:
        cur = con.cursor()
        cur.execute(
            "SELECT id, full_name, grade, letter FROM students WHERE grade=? AND letter=?",
            (grade, letter))
        rows = cur.fetchall()
        if rows:
            students=[]
            for st in rows:
                students.append(students_pb2.Student(
                    id=st[0],
                    full_name=st[1],
                    grade=st[2],
                    letter=st[3]))
            return students
        return None
    
def GetParallelFromDB(grade: int) -> list[students_pb2.Student]:
    with get_connection() as con:
        cur = con.cursor()
        cur.execute(
            "SELECT id, full_name, grade, letter FROM students WHERE grade=?",
            (grade,))
        rows = cur.fetchall()
        if rows:
            students=[]
            for st in rows:
                students.append(students_pb2.Student(
                    id=st[0],
                    full_name=st[1],
                    grade=st[2],
                    letter=st[3]))
            return students
        return None