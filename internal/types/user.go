package types

import "fmt"

type UserInfo struct {
	Sub      string `json:"sub"`            // Identificador único del usuario
	Name     string `json:"name"`           // Nombre completo del usuario
	Nickname string `json:"nickname"`       // Apodo del usuario
	Picture  string `json:"picture"`        // URL de la imagen de perfil del usuario
	Email    string `json:"email"`          // Correo electrónico del usuario
	Verified bool   `json:"email_verified"` // Si el correo electrónico está verificado
}

func (ui UserInfo) String() string {
	return fmt.Sprintf("Name=(%s), email=(%s), nickname=(%s), verified=(%t), sub=(%s)", ui.Name, ui.Email, ui.Nickname, ui.Verified, ui.Sub)
}
