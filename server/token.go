/*

BOOSTER-WEB: Web interface to BOOSTER (https://github.com/evolbioinfo/booster)
Alternative method to compute bootstrap branch supports in large trees.

Copyright (C) 2017 BOOSTER-WEB dev team

This program is free software; you can redistribute it and/or
modify it under the terms of the GNU General Public License
as published by the Free Software Foundation; either version 2
of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program; if not, write to the Free Software
Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.

*/

package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
)

type Key int

const MyKey Key = 0

// If authent == true => then we turn authentication on
var Authent bool = false
var Username string = ""
var Password string = ""
var mySigningKey = []byte(GenerateRandomString(20))

type Claims struct {
	Username string `json:"username"`
	// recommended having
	jwt.StandardClaims
}

type AuthJson struct {
	Username string `json:"username"`
	Password string `json:"password`
}

type AuthResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Token   string `json:"token"`
}

func setToken(res http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	user := req.FormValue("user")
	pass := req.FormValue("pass")
	if user == Username && pass == Password {
		// Expires the token and cookie in 1 hour
		expireToken := time.Now().Add(time.Hour * 1).Unix()
		expireCookie := time.Now().Add(time.Hour * 1)

		// We'll manually assign the claims but in production you'd insert values from a database
		claims := Claims{
			Username,
			jwt.StandardClaims{
				ExpiresAt: expireToken,
				Issuer:    "localhost:8080",
			},
		}

		// Create the token using your claims
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		// Signs the token with a secret.
		signedToken, _ := token.SignedString(mySigningKey)

		// Place the token in the client's cookie
		cookie := http.Cookie{Name: "Auth", Value: signedToken, Expires: expireCookie, HttpOnly: true}
		http.SetCookie(res, &cookie)

		// Redirect the user to root
		http.Redirect(res, req, "/", http.StatusFound)
	} else {
		http.Redirect(res, req, "/login", http.StatusFound)
	}
}

func getToken(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/json")
	body, err := ioutil.ReadAll(req.Body)
	answer := AuthResponse{0, "", ""}
	authjson := AuthJson{}

	if err != nil {
		answer.Status = 1
		answer.Message = err.Error()
	} else if err2 := json.Unmarshal(body, &authjson); err2 != nil {
		answer.Status = 1
		answer.Message = err2.Error()
	} else {
		if authjson.Username == Username && authjson.Password == Password {
			// Expires the token and cookie in 1 hour
			expireToken := time.Now().Add(time.Hour * 10).Unix()

			// We'll manually assign the claims but in production you'd insert values from a database
			claims := Claims{
				Username,
				jwt.StandardClaims{
					ExpiresAt: expireToken,
					Issuer:    "booster.c3bi.pasteur.fr",
				},
			}

			// Create the token using your claims
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

			// Signs the token with a secret.
			signedToken, _ := token.SignedString(mySigningKey)
			answer.Token = signedToken
		} else {
			answer.Status = 1
			answer.Message = "Wrong Credentials"
		}
	}
	if err := json.NewEncoder(res).Encode(answer); err != nil {
		log.Print(err)
	}
}

// Middleware to protect private pages
func validateHtml(page http.HandlerFunc) http.HandlerFunc {
	if Authent {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

			// If no Auth cookie is set then go to login
			cookie, err := req.Cookie("Auth")
			if err != nil {
				http.Redirect(res, req, "/login", http.StatusFound)
				return
			}

			// Return a Token using the cookie
			token, err := jwt.ParseWithClaims(cookie.Value, &Claims{}, func(token *jwt.Token) (interface{}, error) {
				// Make sure token's signature wasn't changed
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected siging method")
				}
				return mySigningKey, nil
			})
			if err != nil {
				http.Redirect(res, req, "/login", http.StatusFound)
				return
			}

			// Grab the tokens claims and pass it into the original request
			if claims, ok := token.Claims.(*Claims); ok && token.Valid {
				ctx := context.WithValue(req.Context(), MyKey, *claims)
				page(res, req.WithContext(ctx))
			} else {
				http.Redirect(res, req, "/login", http.StatusFound)
				return
			}
		})
	} else {
		/* We return a handler without authentication (it does nothing) */
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			page(res, req)
		})
	}
}

// Middleware to protect private API
func validateApi(page http.HandlerFunc) http.HandlerFunc {
	if Authent {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

			var val string
			// If no Auth cookie is set then we look at request bearer auth header
			cookie, err := req.Cookie("Auth")
			if err != nil {
				// Get token from the Authorization header
				// format: Authorization: Bearer
				tokens, ok := req.Header["Authorization"]
				if ok && len(tokens) >= 1 {
					val = tokens[0]
					val = strings.TrimPrefix(val, "Bearer ")
				}
			} else {
				val = cookie.Value
			}

			//fmt.Println(val)

			// Return a Token using the value of the cookie or the bearer
			token, err := jwt.ParseWithClaims(val, &Claims{}, func(token *jwt.Token) (interface{}, error) {
				// Make sure token's signature wasn't changed
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected siging method")
				}
				return mySigningKey, nil
			})
			if err != nil {
				apiError(res, err)
				return
			}

			// Grab the tokens claims and pass it into the original request
			if claims, ok := token.Claims.(*Claims); ok && token.Valid {
				ctx := context.WithValue(req.Context(), MyKey, *claims)
				page(res, req.WithContext(ctx))
			} else {
				apiError(res, errors.New("Problem with authentication token"))
				return
			}
		})
	} else {
		/* We return a handler without authentication (it does nothing) */
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			page(res, req)
		})

	}
}

func protectedProfile(res http.ResponseWriter, req *http.Request) {
	claims, ok := req.Context().Value(MyKey).(Claims)
	if !ok {
		http.Redirect(res, req, "/login", http.StatusFound)
		return
	}

	fmt.Fprintf(res, "Hello %s", claims.Username)
}

func logout(res http.ResponseWriter, req *http.Request) {
	deleteCookie := http.Cookie{Name: "Auth", Value: "none", Expires: time.Now()}
	http.SetCookie(res, &deleteCookie)
	http.Redirect(res, req, "/", http.StatusFound)
	return
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomString(s int) string {
	b, err := GenerateRandomBytes(s)
	if err != nil {
		log.Fatal(err)
	}
	return base64.URLEncoding.EncodeToString(b)
}
