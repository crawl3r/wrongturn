# Wrong Turn  
  
Designed with automation in mind. Send URLs, wait for output. Utilised a file of payloads so can be 'extended'.  
  
### Install

```
go install github.com/crawl3r/wrongturn@latest
```

### Usage  
  
```
cat urls.txt | ./wrongturn -t \"https://redirect.to.me\" -p payloads.txt
```

### Payloads  
  
Thanks to 003random for open sourcing their recon tooling. This tool is basically a copy of theirs but in Go. The payloads I have included are lifted from https://github.com/003random/003Recon/blob/master/payloads/open_redirects.txt, with my added <--key--> for replacing with a custom target URL for the redirect.  
  
To add new payloads, add them on their own lines and just use the <--target--> string in place of the URL (ignoring the protocol). So let's say for example you want the following payload added:  
  
```
?gogo=https://yesplease.com
```
  
You would simply just add:

```
gogo=https://<--target-->
```