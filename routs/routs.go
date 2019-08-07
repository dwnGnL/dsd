package routs

import(
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	// "github.com/jinzhu/gorm"
	"github.com/gorilla/sessions"
	"github.com/dgrijalva/jwt-go"
	"time"
	"fmt"
	"encoding/gob"
	"chat/models"
	"chat/db"
	// "chat/utils"
	"log"
)

var Config *models.Config
func Routs(r *gin.Engine){
	r.LoadHTMLGlob("templates/*")
	session:=r.Group("/ses")
	{
		session.GET("/",Index)
		session.GET("/logout", Logout)
		session.POST("/checkLog", CheckLog)
		session.GET("/ws", func(c *gin.Context) {
			Wshandler(c.Writer, c.Request)
		})
	}
	// r.GET("/", Index)
	// r.GET("/logout", Logout)
	// r.POST("/checkLog", CheckLog)
	// r.GET("/chat",routs.Chat)
	// r.GET("/ws", func(c *gin.Context) {
	// 	Wshandler(c.Writer, c.Request)
	// })
	jwt:=r.Group("/jwt")
	{
		jwt.GET("/",func(c *gin.Context){
			Indexjwt(c,c.Writer, c.Request)
		})
		jwt.GET("/logout", Logout)
		jwt.POST("/checkLog", CheckLogjwt)
		jwt.GET("/ws", func(c *gin.Context) {
			Wshandlerjwt(c.Writer, c.Request)
		})
	}
		go HandleMessages()
}
var cookieStore = sessions.NewCookieStore([]byte("Secret"))

const cookieName = "MyCookie"

type sesKey int

const (
    sesKeyLogin sesKey = iota
)
type User struct {
	Username      string
	Authenticated bool
	Type bool
}


var onlineUsers = make(map[*websocket.Conn]string)
var users = make(map[*websocket.Conn]bool)
var broadcast = make(chan models.Message)
var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}


var jwtKey = []byte("my_secret_key")
type Claims struct {
	Username string `json:"username"`
	Authenticated bool `json:"authenticated"`
	Type bool `json:"type"`
	jwt.StandardClaims
}



// var login string



func CheckCookie(user User) User{
	db := db.GetDB()
	var account []models.Account
	db.Find(&account)
	for _,acc:=range account{
		if user.Username==acc.Login{
			return user
		}
	}
	return User{Authenticated: false}
}

func CheckLog(c *gin.Context){
	ses, err := cookieStore.Get(c.Request, cookieName)
	if err != nil {
		http.Error(c.Writer, err.Error(), http.StatusBadRequest)
		return
	}
	var account []models.Account
	db := db.GetDB()
	login:=c.PostForm("login")
	pass:=c.PostForm("password")
	fmt.Print(login)
	db.Find(&account)
	for _,acc:=range account{
		if login==acc.Login && pass==acc.Pass{
			user := &User{
				Username:      login,
				Authenticated: true,
			}
			ses.Values[sesKeyLogin] = user
			ses.Options.MaxAge=3500
			err = cookieStore.Save(c.Request, c.Writer, ses)
			if err != nil {
				http.Error(c.Writer, err.Error(), http.StatusBadRequest)
				return
			}
		}
	}
		c.Redirect(303,"http://localhost:8080/ses")
}

func CheckLogjwt(c *gin.Context){
	var account []models.Account
	db := db.GetDB()
	login:=c.PostForm("login")
	pass:=c.PostForm("password")
	db.Find(&account)
	for _,acc:=range account{
		if login==acc.Login && pass==acc.Pass{
			logs := models.Logs{User:acc.Login}
			db.Create(&logs)
			expirationTime := time.Now().Add(5 * time.Minute)
			// Create the JWT claims, which includes the username and expiry time
			claims := &Claims{
				Username: login,
				Authenticated:true,
				StandardClaims: jwt.StandardClaims{
					// In JWT, the expiry time is expressed as unix milliseconds
					ExpiresAt: expirationTime.Unix(),
				},
			}
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			// Create the JWT string
			tokenString, err := token.SignedString(jwtKey)
			if err != nil {
				// If there is an error in creating the JWT return an internal server error
				fmt.Print("Failed to set jwtKey")
				return
			}

			// Finally, we set the client cookie for "token" as the JWT we just generated
			// we also set an expiry time which is the same as the token itself
			http.SetCookie(c.Writer, &http.Cookie{
				Name:    "token",
				Value:   tokenString,
				Expires: expirationTime,
			})
		}
	}
	
		c.Redirect(303,"http://localhost:8080/jwt")
	
}

func GetStruct(tknStr string,w http.ResponseWriter)*Claims {
	claims := &Claims{}
	tkn, err := jwt.ParseWithClaims(tknStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			w.WriteHeader(http.StatusUnauthorized)
			return nil
		}
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	if !tkn.Valid {
		w.WriteHeader(http.StatusUnauthorized)
		return nil
	}
	return claims
}

func init(){
	gob.Register(sesKey(0))
	gob.Register(User{})
}

func Index(c *gin.Context){
	ses, err := cookieStore.Get(c.Request, cookieName)
	if err != nil {
		http.Error(c.Writer, err.Error(), http.StatusBadRequest)
		return
	}

	user, _ := ses.Values[sesKeyLogin].(User)

	user=CheckCookie(user)
	user.Type=true
	
	c.HTML(http.StatusOK,"index.gohtml",user)

}


func Indexjwt(c *gin.Context,w http.ResponseWriter, r *http.Request){
	s, err := r.Cookie("token")
	if err != nil{
		expirationTime := time.Now().Add(5 * time.Minute)
		claims := &Claims{
			Username: "Undefind",
			Authenticated:false,
			StandardClaims: jwt.StandardClaims{
				// In JWT, the expiry time is expressed as unix milliseconds
				ExpiresAt: expirationTime.Unix(),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			// Create the JWT string
			tokenString, err := token.SignedString(jwtKey)
			if err != nil {
				// If there is an error in creating the JWT return an internal server error
				fmt.Print("Failed to set jwtKey")
				return
			}

			// Finally, we set the client cookie for "token" as the JWT we just generated
			// we also set an expiry time which is the same as the token itself
			http.SetCookie(c.Writer, &http.Cookie{
				Name:    "token",
				Value:   tokenString,
				Expires: expirationTime,
			})
		c.HTML(http.StatusOK,"index.gohtml",Claims{Authenticated:false})
		return
	}
	// Get the JWT string from the cookie
	tknStr := s.Value

	// Initialize a new instance of `Claims`
	claims := GetStruct(tknStr,w)
	if claims == nil{
		return
	}
	c.HTML(http.StatusOK,"index.gohtml",claims)
}


func Wshandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsupgrader.Upgrade(w, r, nil)
	database:=db.GetDB()
	if err != nil {
		fmt.Printf("Failed to set websocket upgrade: %+v \n", err)
		return
	}
	defer conn.Close()
	ses, err := cookieStore.Get(r, cookieName)
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
	login, _ := ses.Values[sesKeyLogin].(User)
	var history []models.History
	onlineUsers[conn] = login.Username
	users[conn] = true

	database.Find(&history)

	for _, row:= range history{
		historyMsg := models.Message {
			User: row.User,
			Message: row.Message,
			Date: row.Date,
		}
		conn.WriteJSON(historyMsg)
	}

	for {
		var msg models.Message
		// Read in a new message as JSON and map it to a Message object
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(users, conn)
			delete(onlineUsers,conn)
			break
		}
		if msg.Message == "connect" {
			msg.User = onlineUsers[conn]
			msg.Message = "test.conn"
			conn.WriteJSON(msg)
		}else {
			msg.User = onlineUsers[conn]
			// Send the newly received message to the broadcast channel
			broadcast <- msg
		}
	}
}

func Wshandlerjwt(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")
	// Get the JWT string from the cookie
	tknStr := c.Value

	// Initialize a new instance of `Claims`
	claims := GetStruct(tknStr,w)
	if claims == nil{
		return
	}
	conn, err := wsupgrader.Upgrade(w, r, nil)
	database:=db.GetDB()
	if err != nil {
		fmt.Printf("Failed to set websocket upgrade: %+v \n", err)
		return
	}
	defer conn.Close()

	var history []models.History
	onlineUsers[conn] = claims.Username
	users[conn] = true

	database.Find(&history)

	for _, row:= range history{
		historyMsg := models.Message {
			User: row.User,
			Message: row.Message,
			Date: row.Date,
		}
		conn.WriteJSON(historyMsg)
	}

	for {
		var msg models.Message
		// Read in a new message as JSON and map it to a Message object
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(users, conn)
			delete(onlineUsers,conn)
			break
		}
		if msg.Message == "connect" {
			msg.User = onlineUsers[conn]
			msg.Message = "test.conn"
			conn.WriteJSON(msg)
		}else {
			msg.User = onlineUsers[conn]
			// Send the newly received message to the broadcast channel
			broadcast <- msg
		}
	}
}

func Logout(c *gin.Context){
	session, err := cookieStore.Get(c.Request, cookieName)
	if err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	session.Values[sesKeyLogin] = User{}
	session.Options.MaxAge = -1

	err = session.Save(c.Request, c.Writer)
	if err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(c.Writer, c.Request, "/", http.StatusFound)
}

func HandleMessages() {
	database:=db.GetDB()
	now := time.Now().Format("02.01.2006 15:04:05")
	for {
		var history models.History
		// Grab the next message from the broadcast channel
		msg := <-broadcast
		if msg.Message != " is online" {
			history.User = msg.User
			history.Message = msg.Message
			history.Date = now

			database.Create(&history)
		}

		// Send it out to every user that is currently connected
		for user := range users {
			msg.Date = now
			err := user.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				user.Close()
				delete(users, user)
				delete(onlineUsers,user)
			}
		}
	}
}
