package i18n

import "net/http"

func CookieLang(r *http.Request) string {
	if cookie, err := r.Cookie(CookieName); err == nil && IsSupported(cookie.Value) {
		return cookie.Value
	}
	return ""
}

func SetCookie(w http.ResponseWriter, r *http.Request, tag string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    tag,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})
}

func ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
