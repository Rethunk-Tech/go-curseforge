package curseforge

import (
	"errors"
	"net/url"
	"strconv"
)

// Scheme is the scheme of the site's Install button.
const Scheme = "curseforge"

// Ref names one file of a project in an install link.
type Ref struct {
	Project, File int
}

var errBadLink = errors.New("not a curseforge install link")

// ParseLink reads curseforge://install?addonId=<project>&fileId=<file>. Both ids must be positive; any other
// host, a path, userinfo or a fragment is refused. Other query keys are ignored.
func ParseLink(raw string) (Ref, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != Scheme || u.Host != "install" || u.User != nil || u.Fragment != "" ||
		(u.Path != "" && u.Path != "/") {
		return Ref{}, errBadLink
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return Ref{}, errBadLink
	}
	project, perr := strconv.Atoi(q.Get("addonId"))
	file, ferr := strconv.Atoi(q.Get("fileId"))
	if perr != nil || ferr != nil || project <= 0 || file <= 0 {
		return Ref{}, errBadLink
	}
	return Ref{Project: project, File: file}, nil
}
