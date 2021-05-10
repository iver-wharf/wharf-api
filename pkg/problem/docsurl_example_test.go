package problem

import (
	"fmt"
	"net/url"
)

func ExampleConvertURLToAbsDocsURL() {
	var u *url.URL

	u, _ = url.Parse("https://iver-wharf.github.io/#/prob/build/run/invalid-input")
	fmt.Println("Unaltered 1:", ConvertURLToAbsDocsURL(*u).String() == u.String())

	u, _ = url.Parse("http://some-other-page/prob/build/run/invalid-input")
	fmt.Println("Unaltered 2:", ConvertURLToAbsDocsURL(*u).String() == u.String())

	u, _ = url.Parse("https://iver-wharf.github.io/prob/build/run/invalid-input")
	fmt.Println("Fragmented path:", ConvertURLToAbsDocsURL(*u).String())

	u, _ = url.Parse("/prob/build/run/invalid-input")
	fmt.Println("Added schema & host:", ConvertURLToAbsDocsURL(*u).String())

	u, _ = url.Parse("/prob/build/run/invalid-input")
	fmt.Println("Leaves original intact:", ConvertURLToAbsDocsURL(*u).String() != u.String())

	// Output:
	// Unaltered 1: true
	// Unaltered 2: true
	// Fragmented path: https://iver-wharf.github.io/#/prob/build/run/invalid-input
	// Added schema & host: https://iver-wharf.github.io/#/prob/build/run/invalid-input
	// Leaves original intact: true
}
