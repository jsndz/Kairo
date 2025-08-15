package service

import (
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jsndz/kairo/apps/auth-service/internal/app/model"
	repos "github.com/jsndz/kairo/apps/auth-service/internal/app/repo"
	"github.com/jsndz/kairo/apps/auth-service/utils"
	"gorm.io/gorm"
)
var jwtSecret = os.Getenv("JWT_SECRET")
var wsjwtSecret = os.Getenv("WS_JWT_SECRET")

type UserService struct {
	userRepo   *repos.UserRepository
}

func NewUserService (db *gorm.DB) * UserService {
	return &UserService{
		userRepo: repos.NewUserRepository(db),
	}
}


func (s *UserService) Signup(data model.User)(string,*model.User,error){
	user,err := s.userRepo.Create(&data);
	if  err != nil {
		return "",nil,err
	}
	jwt_token, err := utils.GenerateJWT(data.Email,user.ID)
	if  err != nil {
		return "",nil,err
	}
	return jwt_token,user,nil
}

func (s *UserService) Signin(Email string,Password string)(string,*model.User,error){
	var user *model.User
	user, err := s.userRepo.Get(Email)
	if  err != nil {
		return "", nil,fmt.Errorf("user doesn't exist")
	}

	if !model.CheckPassword(user.Password,Password) {
		return "", nil,fmt.Errorf("invalid credentials")
	}

	jwt_token, err := utils.GenerateJWT(Email,user.ID)

	if  err != nil {
		return "",nil,err
	}
	return jwt_token,user,nil
}

func (s *UserService) Authenticate(token string) (string, error) {

	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil || !parsedToken.Valid {
		return "", fmt.Errorf("invalid token")
	}
	 claims, ok := parsedToken.Claims.(jwt.MapClaims);
	 if !ok {
	}
	userid, ok := claims["id"].(string)
	if !ok {
		return "", fmt.Errorf("userid is not a string")
	}
	return userid, nil
}


func (s *UserService) GetName(ID uint)(string,error){
	return s.userRepo.GetName(ID)
}

func (s *UserService) CreateWSToken(docId uint32,userId uint32)(string,error){
	token,err:= utils.GenerateJWTForWS(docId,userId)
	if err!=nil{
		return "",err
	}
	return token,err
}

func (s *UserService) AuthenticateWS(token string) (string,string, error) {

	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(wsjwtSecret), nil
	})
	if err != nil || !parsedToken.Valid {
		return "","",  fmt.Errorf("invalid token")
	}
	 claims, ok := parsedToken.Claims.(jwt.MapClaims);
	 if !ok {
	}
	userid, ok := claims["sub"].(string)
	if !ok {
		return "","", fmt.Errorf("userid is not a string")
	}
	docId, ok := claims["doc_id"].(string)
	if !ok {
		return "","",  fmt.Errorf("userid is not a string")
	}
	return userid,docId, nil
}
