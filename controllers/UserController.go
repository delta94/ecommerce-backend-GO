package controller

import (
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber"
	"github.com/gofrs/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/ecommerce/db"
	u "github.com/ecommerce/models"
	q "github.com/ecommerce/utility"
)

type login struct {
	Email		string	`json:"email"`
	Password	string	`json:"password"`
}

type claims struct {
	email	string
	jwt.StandardClaims
}

//SignUpUser create user in db
func SignUpUser(w *fiber.Ctx)  {
	id, err := uuid.NewV4()
	if err != nil {
		w.Status(500).JSON("Error in uuid generate")
		return
	}

	var aux u.User
	aux.ID = id
	aux.Name = w.FormValue("name")
	aux.Password = w.FormValue("password")
	aux.Email = w.FormValue("email")
	aux.Phone, err = strconv.Atoi(w.FormValue("phone"))
	if err != nil {
		w.Status(500).JSON("Error in parse form")
		return
	}
	aux.Address = w.FormValue("address")
	aux.FacebookID = w.FormValue("facebookID")
	aux.Birthday = w.FormValue("birthday")
	aux.Gender = w.FormValue("gender")

	var aux1 u.User
	db.DBConn.Where("email = ?", aux.Email).First(&aux1)
	if len(aux1.Email) != 0 {
		w.Status(400).JSON("Already has a user with this email")
		return
	}

	hashPassword, err := bcrypt.GenerateFromPassword([]byte(aux.Password), bcrypt.DefaultCost)
	if err != nil {
		w.Status(500).JSON("Error in bcrypt")
		return
	}
	aux.Password = string(hashPassword)

	avatarFile, _ := w.FormFile("avatar")
	if avatarFile != nil {
		file, err := avatarFile.Open()
		avatarResponse := q.SendImageToAWS(file, avatarFile.Filename, avatarFile.Size, "user")
		if avatarResponse == "Error upload image to AWS" || err != nil {
			w.Status(500).JSON(avatarResponse)
			return
		}
		defer file.Close()
		aux.Avatar = avatarResponse
	} else {
		aux.Avatar = ""
	}

	result := db.DBConn.Create(&aux)
	if result.Error != nil {
		w.Status(403).JSON("Error creating user")
		return
	}

	w.Status(201).JSON("User created")
}

//Login user in application
func Login(w *fiber.Ctx) {
	login := new(login)
	if err := w.BodyParser(login); err != nil {
		w.Status(403).JSON("Missing fields")
		return
	}

	var user u.User
	db.DBConn.Where("email = ?", login.Email).Find(&user)
	if user.Phone == 0 {
		w.Status(403).JSON("No user with this email")
		return
	}

	hashPass := []byte(user.Password)
	bodyPass := []byte(login.Password)
	errorHash := bcrypt.CompareHashAndPassword(hashPass, bodyPass)
	if errorHash != nil {
		w.Status(403).JSON("Wrong password")
		return
	}

	expTime := time.Now().Add(4000 * time.Minute)
	claimsJwt := &claims{
		email: login.Email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expTime.Unix(),
		},
	}
	tokenMethod := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsJwt)
	jwtKey := []byte(q.GetDotEnv("JWT_KEY"))
	token, err := tokenMethod.SignedString(jwtKey)
	if err != nil {
		w.Status(500).JSON("Error in jwt token")
		return
	}

	w.Status(200).JSON(&fiber.Map{
		"user": user,
		"token": token,
	})
}