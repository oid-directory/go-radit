package jii

/*
github actions fails to build any package that contains only
variables (such as this package). Thus we'll just add an empty
func to work around this lame situation.
*/
func satisfyGithubActions() {}
