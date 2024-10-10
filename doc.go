/*
Package radit wraps [radir] to provide a convenient directory information
tree abstraction and content parser/generator to assist in the creation
of an OID Directory tree.

At present, this package can produce an LDIF (text) dump that contains
well over 125000 entries, comprised of both *[radir.Registration] and
*[radir.Registrant] instances.

See the [RADIT I-D] for details.

Please note this is a very early release; breaking changes are likely!

[radir]: https://github.com/oid-directory/go-radir
[RADIT I-D]: https://datatracker.ietf.org/doc/html/draft-coretta-oiddir-radit
*/
package radit
