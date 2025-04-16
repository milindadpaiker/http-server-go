# HTTP server implementation in GO. Supports::
1. File server. Root folder can be provided during server init (see app/main.go) or as --directory "./path1/path2" flag when starting main application
2. Response body compression using gzip
3. Persistent connections