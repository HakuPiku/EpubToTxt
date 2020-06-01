# EpubToTxt

Gets all the text content from an epub and saves it as txt. 

To run it :

`go run main.go -epub=<epub file> -regex=<regex file>`


A regex file can be added to to replace certain parts of the epub content. 

The i-th(first) line in the regex file defines the regex to match and i+1th(second) line defines what to replace the matched regex with.

I wanted to remove everything between `<rt>` tags because they mess up my text parsing software so my regex file looks like this:
```
<rt>(.*?)</rt>
 
```
