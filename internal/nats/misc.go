package nats

import (
	"slices"
	"strings"
)

/* If fqdn is "www.example.com", output will be "prefix.com.example.www.suffix" */
func getSubjectFromFqdn(natsPrefix, fqdn, natsSuffix string) string {
	fqdnSplit := slices.DeleteFunc(strings.Split(fqdn, "."),
		func(s string) bool { return s == "" })

	if natsPrefix != "" {
		fqdnSplit = append(fqdnSplit, natsPrefix)
	}

	if natsSuffix != "" {
		fqdnSplit = append([]string{natsSuffix}, fqdnSplit...)
	}

	slices.Reverse(fqdnSplit)

	return strings.Join(fqdnSplit, ".")
}
