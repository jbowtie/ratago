<?xml version="1.0" encoding="UTF-8" ?>
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<xsl:param name="numberVal" as="xs:integer" />
<xsl:param name="stringVal" as="xs:string" />

<xsl:template match="/">
  <body>
	  <number><xsl value-of="$numberVal + 1" /></number>
	  <text><xsl value-of="$stringVal" /></text>
  </body>
</xsl:template>

</xsl:stylesheet>
