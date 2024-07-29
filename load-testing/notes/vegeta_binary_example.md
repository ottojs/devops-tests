# Vegeta Binary Example

This is fine for simple tests but using the library is more flexible
`echo "GET http://localhost" | vegeta attack -duration=120s -rate=10 | tee results.bin | vegeta report;`
