package i18n

import (
	"net/http"

	"golang.org/x/text/language"
)

var matcher = language.NewMatcher(supportedTags())

func supportedTags() []language.Tag {
	tags := make([]language.Tag, len(Supported))
	for i, l := range Supported {
		tags[i] = language.MustParse(l.Tag)
	}
	return tags
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Accept-Language, Cookie")
		next.ServeHTTP(w, r.WithContext(WithLang(r.Context(), resolve(r))))
	})
}

func resolve(r *http.Request) string {
	if cookie, err := r.Cookie(CookieName); err == nil && IsSupported(cookie.Value) {
		return cookie.Value
	}

	preferred, _, _ := language.ParseAcceptLanguage(r.Header.Get("Accept-Language"))
	_, index, _ := matcher.Match(preferred...)
	return Supported[index].Tag
}
