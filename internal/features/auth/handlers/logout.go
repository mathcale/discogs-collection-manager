package handlers

import (
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/usecases"
)

func (h *Handler) LogoutSubmit(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil {
		h.LogoutUseCase.Execute(
			r.Context(),
			usecases.LogoutInput{
				SessionToken: cookie.Value,
			},
		)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.Redirect(w, r, "/auth/login", http.StatusFound)
}
