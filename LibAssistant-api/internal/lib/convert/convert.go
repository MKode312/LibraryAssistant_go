package convert

import (
	"LibAssistant_api/internal/domain/models"
	"strings"

	studentsv1 "github.com/MKode312/protos/gen/go/LibAssistant/students"
)

func StudentToDomain(protoStudent *studentsv1.Student) models.Student {
	return models.Student{
		ID:       protoStudent.GetId(),
		FullName: protoStudent.GetFullName(),
		Grade:    protoStudent.GetGrade(),
		Letter:   protoStudent.GetLetter(),
	}
}

func StudentsServiceErrorToHTTPResponseError(err error) string {
	strErr := err.Error()
	firstIdx := strings.Index(strErr, ":")
	if firstIdx == -1 {
		return strErr
	}
	secondIdx := strings.Index(strErr[firstIdx+1:], ":")
	if secondIdx == -1 {
		return strErr
	}
	secondIdx += firstIdx + 1

	return strErr[secondIdx+1:]
}
