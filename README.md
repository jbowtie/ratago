ratago
======

Ratago is a (mostly-compliant) implementation of an XSLT 1.0 processor written in Go and released under an MIT license.

Currently it should be seen as experimental - it lacks full compliance with spec and has only been used to run simple test scripts.

The test suite is derived from the test suite used by the libxslt library written by Daniel Veillard. See http://xmlsoft.org/XSLT/ for details on libxslt.

TODO
----

There are several tasks remaining to reach full compliance. Until these tasks are complete the API is subject to change.

* Fully implement xsl:include (see tests in libxslt general suite)
* Fully implement xsl:message (no tests in initial suite)
* Ensure that errors are properly progogated in Go fashion.

