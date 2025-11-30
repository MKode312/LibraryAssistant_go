package tests

import (
	"LibAssistant_sso/tests/suite"
	"testing"

	ssov1 "github.com/MKode312/protos/gen/go/LibAssistant/sso"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterAsAdmin_DuplicateRegistrationAsAdmin(t *testing.T) {
	ctx, st := suite.New(t)

	email := gofakeit.Email()
	pass := randomFakePassword()

	respReg, err := st.AuthClient.RegisterAsAdmin(ctx, &ssov1.RegisterAsAdminRequest{
		Email:    email,
		Password: pass,
		AdminSecret: adminSecret,
	})

	require.NoError(t, err)
	require.NotEmpty(t, respReg.GetUserId())

	respReg, err = st.AuthClient.RegisterAsAdmin(ctx, &ssov1.RegisterAsAdminRequest{
		Email:    email,
		Password: pass,
		AdminSecret: adminSecret,
	})
	require.Error(t, err)
	assert.Empty(t, respReg.GetUserId())
	assert.ErrorContains(t, err, "user already exists")
}

func TestRegisterAsAdmin_WrongAdminSecret(t *testing.T) {
	ctx, st := suite.New(t)

	email := gofakeit.Email()
	password := randomFakePassword()
	adminSecret := randomFakeAdminKey()

	respRegAsAdmin, err := st.AuthClient.RegisterAsAdmin(ctx, &ssov1.RegisterAsAdminRequest{
		Email: email,
		Password: password,
		AdminSecret: adminSecret,
	})

	require.Error(t, err)
	assert.Empty(t, respRegAsAdmin.GetUserId())
	assert.ErrorContains(t, err, "wrong admin secret key")
}

func TestIsAdmin_HappyPath(t *testing.T) {
	ctx, st := suite.New(t)
	
	email := gofakeit.Email()
	password := randomFakePassword()

	respRegAsAdmin, err := st.AuthClient.RegisterAsAdmin(ctx, &ssov1.RegisterAsAdminRequest{
		Email: email,
		Password: password,
		AdminSecret: adminSecret,
	})

	require.NoError(t, err)
	require.NotEmpty(t, respRegAsAdmin.GetUserId())

	respIsAdmin, err := st.AuthClient.IsAdmin(ctx, &ssov1.IsAdminRequest{
		UserId: respRegAsAdmin.GetUserId(),
	})

	require.NoError(t, err)
	assert.NotEmpty(t, respIsAdmin.GetIsAdmin())
}

func TestIsAdmin_False(t *testing.T) {
	ctx, st := suite.New(t)

	email := gofakeit.Email()
	password := randomFakePassword()

	respReg, err := st.AuthClient.Register(ctx, &ssov1.RegisterRequest{
		Email: email,
		Password: password,
	})

	require.NoError(t, err)
	require.NotEmpty(t, respReg.GetUserId())

	respIsAdmin, err := st.AuthClient.IsAdmin(ctx, &ssov1.IsAdminRequest{
		UserId: respReg.GetUserId(),
	})

	require.NoError(t, err)
	require.Equal(t, false, respIsAdmin.GetIsAdmin())
}

func TestIsAdmin_True(t *testing.T) {
	ctx, st := suite.New(t)

	email := gofakeit.Email()
	password := randomFakePassword()
	
	respRegAsAdmin, err := st.AuthClient.RegisterAsAdmin(ctx, &ssov1.RegisterAsAdminRequest{
		Email: email,
		Password: password,
		AdminSecret: adminSecret,
	})

	require.NoError(t, err)
	require.NotEmpty(t, respRegAsAdmin.GetUserId())

	respIsAdmin, err := st.AuthClient.IsAdmin(ctx, &ssov1.IsAdminRequest{
		UserId: respRegAsAdmin.GetUserId(),
	})

	require.NoError(t, err)
	require.Equal(t, true, respIsAdmin.GetIsAdmin())

}

func TestIsAdmin_NotFound(t *testing.T) {
	ctx, st := suite.New(t)

	userID := randomFakeID()

	respIsAdmin, err := st.AuthClient.IsAdmin(ctx, &ssov1.IsAdminRequest{
		UserId: userID,
	})

	require.Error(t, err)
	assert.Empty(t, respIsAdmin.GetIsAdmin())
	assert.ErrorContains(t, err, "user not found")
}

func randomFakeAdminKey() string {
	return gofakeit.RandomString([]string{"Hello", "World", "Golang", "Java", "Python", "C++", "JavaScript"})
}