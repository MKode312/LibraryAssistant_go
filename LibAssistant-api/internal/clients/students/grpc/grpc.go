package studentsgrpc

import (
	"LibAssistant_api/internal/domain/models"
	"LibAssistant_api/internal/lib/convert"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	studentsv1 "github.com/MKode312/protos/gen/go/LibAssistant/students"
	grpclog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Client struct {
	api studentsv1.StudentsServiceClient
	log *slog.Logger
}

var (
	ErrInvalidRequest   = errors.New("invalid request")
	ErrInternal         = errors.New("internal error")
	ErrStudentExists    = errors.New("student already exists")
	ErrStudentNotFound  = errors.New("student not found")
	ErrClassNotFound    = errors.New("class not found")
	ErrParallelNotFound = errors.New("parallel not found")
)

func New(ctx context.Context, log *slog.Logger, addr string, timeout time.Duration, retriesCount int) (*Client, error) {
	const op = "students.grpc.New"

	retryOpts := []grpcretry.CallOption{
		grpcretry.WithCodes(codes.NotFound, codes.Aborted, codes.DeadlineExceeded),
		grpcretry.WithMax(uint(retriesCount)),
		grpcretry.WithPerRetryTimeout(timeout),
	}

	logOpts := []grpclog.Option{
		grpclog.WithLogOnEvents(grpclog.PayloadReceived, grpclog.PayloadSent),
	}

	cc, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			grpclog.UnaryClientInterceptor(interceptorLogger(log), logOpts...),
			grpcretry.UnaryClientInterceptor(retryOpts...),
		))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Client{
		api: studentsv1.NewStudentsServiceClient(cc),
		log: log,
	}, nil
}

func (c *Client) CreateStudent(ctx context.Context, fullName string, grade int32, letter string, studentID string) (string, error) {
	const op = "students.grpc.CreateStudent"

	resp, err := c.api.CreateStudent(ctx, &studentsv1.CreateStudentRequest{
		FullName:  fullName,
		Grade:     grade,
		Letter:    letter,
		StudentId: studentID,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return "", fmt.Errorf("%s: %w: %s", op, ErrInvalidRequest, st.Message())
			}
			if st.Code() == codes.AlreadyExists {
				return "", fmt.Errorf("%s: %w", op, ErrStudentExists)
			}
			if st.Code() == codes.Internal {
				return "", fmt.Errorf("%s: %w", op, ErrInternal)
			}
		}
		return "", fmt.Errorf("%s: %w", op, ErrInternal)
	}

	return resp.GetStudentId(), nil
}

func (c *Client) GetStudentByID(ctx context.Context, studentID string) (models.Student, error) {
	const op = "students.grpc.GetStudentByID"

	resp, err := c.api.GetStudentByID(ctx, &studentsv1.GetStudentByIDRequest{
		StudentId: studentID,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return models.Student{}, fmt.Errorf("%s: %w: %s", op, ErrInvalidRequest, st.Message())
			}
			if st.Code() == codes.NotFound {
				return models.Student{}, fmt.Errorf("%s: %w", op, ErrStudentNotFound)
			}
			if st.Code() == codes.Internal {
				return models.Student{}, fmt.Errorf("%s: %w", op, ErrInternal)
			}
		}
		return models.Student{}, fmt.Errorf("%s: %w", op, ErrInternal)
	}

	student := convert.StudentToDomain(resp.GetStudent())

	return student, nil
}

func (c *Client) GetStudentByFullName(ctx context.Context, fullName string) (models.Student, error) {
	const op = "students.grpc.GetStudentByFullName"

	resp, err := c.api.GetStudentByFullName(ctx, &studentsv1.GetStudentByFullNameRequest{
		FullName: fullName,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return models.Student{}, fmt.Errorf("%s: %w: %s", op, ErrInvalidRequest, st.Message())
			}
			if st.Code() == codes.NotFound {
				return models.Student{}, fmt.Errorf("%s: %w", op, ErrStudentNotFound)
			}
			if st.Code() == codes.Internal {
				return models.Student{}, fmt.Errorf("%s: %w", op, ErrInternal)
			}
		}
		return models.Student{}, fmt.Errorf("%s: %w", op, ErrInternal)
	}

	student := convert.StudentToDomain(resp.GetStudent())

	return student, nil
}

func (c *Client) GetClass(ctx context.Context, grade int32, letter string) ([]models.Student, error) {
	const op = "students.grpc.GetGrade"

	resp, err := c.api.GetClass(ctx, &studentsv1.GetClassRequest{
		Grade:  grade,
		Letter: letter,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return nil, fmt.Errorf("%s: %w: %s", op, ErrInvalidRequest, st.Message())
			}
			if st.Code() == codes.NotFound {
				return nil, fmt.Errorf("%s: %w", op, ErrClassNotFound)
			}
			if st.Code() == codes.Internal {
				return nil, fmt.Errorf("%s: %w", op, ErrInternal)
			}
		}
		return nil, fmt.Errorf("%s: %w", op, ErrInternal)
	}

	var class []models.Student

	for _, protoStudent := range resp.GetStudents() {
		student := convert.StudentToDomain(protoStudent)
		class = append(class, student)
	}

	return class, nil
}

func (c *Client) GetParallel(ctx context.Context, grade int32) ([]models.Student, error) {
	const op = "students.grpc.GetParallel"

	resp, err := c.api.GetParallel(ctx, &studentsv1.GetParallelRequest{
		Grade: grade,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return nil, fmt.Errorf("%s: %w: %s", op, ErrInvalidRequest, st.Message())
			}
			if st.Code() == codes.NotFound {
				return nil, fmt.Errorf("%s: %w", op, ErrParallelNotFound)
			}
			if st.Code() == codes.Internal {
				return nil, fmt.Errorf("%s: %w", op, ErrInternal)
			}
		}
		return nil, fmt.Errorf("%s: %w", op, ErrInternal)
	}

	var parallel []models.Student

	for _, protoStudent := range resp.GetStudents() {
		student := convert.StudentToDomain(protoStudent)
		parallel = append(parallel, student)
	}

	return parallel, nil
}

//TODO: implement GetStats

func (c *Client) UpdateFullNameOfStudent(ctx context.Context, newFullName string, studentID string) (models.Student, error) {
	const op = "students.grpc.UpdateFullNameOfStudent"

	resp, err := c.api.UpdateFullNameOfStudent(ctx, &studentsv1.UpdateFullNameOfStudentRequest{
		FullName: newFullName,
		StudentId: studentID,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return models.Student{}, fmt.Errorf("%s: %w: %s", op, ErrInvalidRequest, st.Message())
			}
			if st.Code() == codes.NotFound {
				return models.Student{}, fmt.Errorf("%s: %w", op, ErrStudentNotFound)
			}
			if st.Code() == codes.Internal {
				return models.Student{}, fmt.Errorf("%s: %w", op, ErrInternal)
			}
		}
		return models.Student{}, fmt.Errorf("%s: %w", op, ErrInternal)
	}

	student := convert.StudentToDomain(resp.GetStudent())

	return student, nil
}

func (c *Client) UpdateGradeOfStudent(ctx context.Context, newGrade int32, studentID string) (models.Student, error) {
	const op = "students.grpc.UpdateFullNameOfStudent"

	resp, err := c.api.UpdateGradeOfStudent(ctx, &studentsv1.UpdateGradeOfStudentRequest{
		Grade: newGrade,
		StudentId: studentID,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return models.Student{}, fmt.Errorf("%s: %w: %s", op, ErrInvalidRequest, st.Message())
			}
			if st.Code() == codes.NotFound {
				return models.Student{}, fmt.Errorf("%s: %w", op, ErrStudentNotFound)
			}
			if st.Code() == codes.Internal {
				return models.Student{}, fmt.Errorf("%s: %w", op, ErrInternal)
			}
		}
		return models.Student{}, fmt.Errorf("%s: %w", op, ErrInternal)
	}

	student := convert.StudentToDomain(resp.GetStudent())

	return student, nil
}

func (c *Client) UpdateLetterOfStudent(ctx context.Context, newLetter string, studentID string) (models.Student, error) {
	const op = "students.grpc.UpdateFullNameOfStudent"

	resp, err := c.api.UpdateLetterOfStudent(ctx, &studentsv1.UpdateLetterOfStudentRequest{
		Letter: newLetter,
		StudentId: studentID,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return models.Student{}, fmt.Errorf("%s: %w: %s", op, ErrInvalidRequest, st.Message())
			}
			if st.Code() == codes.NotFound {
				return models.Student{}, fmt.Errorf("%s: %w", op, ErrStudentNotFound)
			}
			if st.Code() == codes.Internal {
				return models.Student{}, fmt.Errorf("%s: %w", op, ErrInternal)
			}
		}
		return models.Student{}, fmt.Errorf("%s: %w", op, ErrInternal)
	}

	student := convert.StudentToDomain(resp.GetStudent())

	return student, nil
}

func (c *Client) DeleteStudentByID(ctx context.Context, studentID string) (bool, error) {
	const op = "students.grpc.DeleteStudentByID"

	resp, err := c.api.DeleteStudentByID(ctx, &studentsv1.DeleteStudentByIDRequest{
		StudentId: studentID,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.InvalidArgument {
				return false, fmt.Errorf("%s: %w: %s", op, ErrInvalidRequest, st.Message())
			}
			if st.Code() == codes.NotFound {
				return false, fmt.Errorf("%s: %w", op, ErrStudentNotFound)
			}
			if st.Code() == codes.Internal {
				return false, fmt.Errorf("%s: %w", op, ErrInternal)
			}
		}
		return false, fmt.Errorf("%s: %w", op, ErrInternal)
	}

	return resp.GetSuccess(), nil
}

func interceptorLogger(l *slog.Logger) grpclog.Logger {
	return grpclog.LoggerFunc(func(ctx context.Context, level grpclog.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(level), msg, fields...)
	})
}
