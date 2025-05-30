package server

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/jafari-mohammad-reza/dotsync/pkg"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/bcrypt"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}
func InitHttpServer() error {
	server := echo.New()
	server.Validator = &CustomValidator{validator: validator.New()}
	server.Use(middleware.Logger())
	server.Use(middleware.Recover())
	api := server.Group("/api")
	api.POST("/invoke-token", invokeToken)
	api.GET("/revoke-token", revokeToken)
	api.GET("/upload-list", uploadList)
	return server.Start(fmt.Sprintf(":%d", cfg.HttpPort))
}
func validateToken(token string) (string, error) {
	claims, err := pkg.DecodeToken(token)
	if err != nil {
		return "", err
	}
	email := claims["email"].(string)
	if email == "" {
		return "", errors.New("invalid token")
	}
	return email, nil
}
func uploadList(c echo.Context) error {
	token := c.Request().Header.Get("Authorization")
	email, err := validateToken(token)
	if err != nil {
		return c.JSON(401, map[string]interface{}{
			"message": fmt.Sprintf("invalid token %s", err.Error()),
		})
	}
	uploads, err := getUserUploads(email)
	if err != nil {
		return c.JSON(500, map[string]interface{}{
			"message": err.Error(),
		})
	}
	return c.JSON(200, uploads)
}
func invokeToken(c echo.Context) error {
	var body pkg.InvokeBody
	if err := c.Bind(&body); err != nil {
		return c.JSON(400, map[string]interface{}{
			"message": "invalid request",
		})
	}
	if err := c.Validate(&body); err != nil {
		return c.JSON(400, map[string]interface{}{
			"message": "invalid request",
		})
	}
	aget := c.Request().Header.Get("User-Agent")
	body.Agent = aget
	existUser, err := findUser(body.Email)
	if err != nil {
		return c.JSON(500, map[string]interface{}{
			"message": "internal server error",
		})
	}
	if existUser == nil {
		hashPass, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if err != nil {
			return c.JSON(500, map[string]interface{}{
				"message": "internal server error",
			})
		}
		if err := createUser(body.Email, body.Agent, string(hashPass)); err != nil {
			return c.JSON(500, map[string]interface{}{
				"message": "internal server error",
			})
		}
	} else {
		if err := updateAgents(existUser.Email, body.Agent); err != nil {
			return c.JSON(500, map[string]interface{}{
				"message": "internal server error",
			})
		}
	}
	token, err := pkg.GenerateApiKey(body.Email, body.Agent)
	if err != nil {
		fmt.Errorf("generate tojen err %v", err)
		return c.JSON(500, map[string]interface{}{
			"message": "internal server error",
		})
	}
	return c.JSON(200, map[string]string{"token": token})
}
func revokeToken(c echo.Context) error {
	token := c.Request().Header.Get("Authorization")
	if token == "" {
		return c.JSON(400, map[string]interface{}{
			"message": "invalid request",
		})
	}
	claims, err := pkg.DecodeToken(token)
	if err != nil {
		return c.JSON(500, map[string]interface{}{
			"message": "internal server error",
		})
	}
	email := claims["email"].(string)
	agent := claims["agent"].(string)
	foundUser, err := findUser(email)
	if err != nil {
		return err
	}
	if foundUser == nil {
		return c.JSON(400, map[string]interface{}{
			"message": "invalid request",
		})
	}
	if err := deleteAgent(email, agent); err != nil {
		return c.JSON(500, map[string]interface{}{
			"message": "internal server error",
		})
	}

	return c.JSON(200, map[string]string{"message": "token revoked"})
}
