HTTP-Proxy-scanner
===========

HTTP Proxy scan for web spider like eh.mengsky.net

In E5 2660 1 processor with 10K connection it spent 20% CPU 20K/R.

### Add a scan task through beanstalk

`beanstalk.put(json.dumps({"ip":"55.11.22.33", "port":80}))`


