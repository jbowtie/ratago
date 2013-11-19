<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns="http://example.org">

    <xsl:template match="/">
        <html>
            <body>
                <xsl:apply-templates />
            </body>
        </html>
    </xsl:template>

    <xsl:template match="foo">
        <h3><xsl:text>(TITLE) </xsl:text><xsl:value-of select="@title" /></h3>
        <xsl:if test="@title">
            <xsl:apply-templates select="bar[1]" mode="correct" />
        </xsl:if>
    </xsl:template>

    <xsl:template match="bar">
        FAIL template mode
    </xsl:template>

    <xsl:template match="bar" mode="correct" priority="2">
        <p><xsl:value-of select="."/></p>
    </xsl:template>

    <xsl:template match="bar" mode="correct">
        FAIL template priority
    </xsl:template>

</xsl:stylesheet>
