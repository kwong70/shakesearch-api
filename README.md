# API for Pulley's Shakesearch take home project  
## Related Frontend: https://github.com/kwong70/shakesearch
## Render.com server: https://kw-shakesearch-api.onrender.com/

## Endpoint: 

#### /search?q=<string>?exactMatch<bool>
- q : string - the query to search texts 
- exactMatch : boolean - flag to search for exact match. Used if searching for phrases and want to look up the phrase exactly or each word in the phrase separatly.
- body : [] -  list of  titles search q under. If empty searches all titles 

## Details
#### Port used for local: 3001  
#### Run on Development/Local: 
```
go run main.go
```
