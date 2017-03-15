FROM alpine
MAINTAINER bo.yang@dell.com

ADD ecsbeat_linux_amd64 /ecsbeat/ecsbeat
ADD ecsbeat.yml.target /ecsbeat/ecsbeat.yml
ADD ecsbeat.template.json /ecsbeat/
ADD ecsbeat.template-es2x.json /ecsbeat/

WORKDIR /ecsbeat/

CMD ["./ecsbeat", "-e"]
