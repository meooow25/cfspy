package fetch

import urlpkg "net/url"

func removeLocaleParam(url string) (string, error) {
	u, err := urlpkg.Parse(url)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Del("locale")
	u.RawQuery = q.Encode()
	return u.String(), nil
}
