FROM ruby:alpine
RUN apk add --update imagemagick build-base sqlite-dev
RUN echo "gem: --no-rdoc --no-ri" > /etc/gemrc
RUN addgroup massikone && adduser -D -h /massikone -G massikone massikone
ENV BUNDLE_WITHOUT=mysql:pg
ENV RACK_ENV=deployment
ENV DATABASE_URL=sqlite:///data/massikone.sqlite
ADD massikone-docker.tgz /massikone/
RUN (cd /massikone && bundle install)
RUN mkdir /data && chown massikone:massikone /data
VOLUME /data
WORKDIR /massikone
USER massikone
EXPOSE 5000
CMD puma -p 5000
