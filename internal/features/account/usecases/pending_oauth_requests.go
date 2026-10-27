package usecases

import "sync"

type pendingOAuthRequest struct {
	secret string
	userID string
}

type PendingOAuthRequests struct {
	m sync.Map
}

func (p *PendingOAuthRequests) Store(requestToken, secret, userID string) {
	p.m.Store(requestToken, pendingOAuthRequest{secret: secret, userID: userID})
}

func (p *PendingOAuthRequests) Delete(requestToken string) {
	p.m.Delete(requestToken)
}

func (p *PendingOAuthRequests) Take(requestToken string) (secret, userID string, ok bool) {
	value, ok := p.m.LoadAndDelete(requestToken)
	if !ok {
		return "", "", false
	}

	pending, _ := value.(pendingOAuthRequest)

	return pending.secret, pending.userID, true
}
