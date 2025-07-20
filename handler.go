package main

import (
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

func Handler() func(w radius.ResponseWriter, r *radius.Request) {
	return func(w radius.ResponseWriter, r *radius.Request) {
		username := rfc2865.UserName_GetString(r.Packet)
		fullPassword := rfc2865.UserPassword_GetString(r.Packet)
		if len(fullPassword) < 6 && len(username) < 4 && len(username) > 20 && len(fullPassword) > 40 {
			err := w.Write(r.Response(radius.CodeAccessReject))
			if err != nil {
				return
			}
			return
		}
		valid, otp, code := ServiceInstance.ValidLdapCredencials(username, fullPassword)
		if !valid {
			err := w.Write(r.Response(code))
			if err != nil {
				return
			}
			return
		}
		if !ServiceInstance.ValidTotp(otp, username) {
			code = radius.CodeAccessReject
		}
		err := w.Write(r.Response(code))
		if err != nil {
			return
		}
		return
	}
}
