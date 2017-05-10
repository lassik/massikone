FROM ruby:2.2.7-alpine

VOLUME /massikone/data
EXPOSE 5000

ENV BUNDLE_WITHOUT=mysql:pg
ENV RACK_ENV=deployment

COPY . /massikone
WORKDIR /massikone
RUN apk update \
    && apk add \
        imagemagick \
        sqlite \
        sqlite-libs \
    && apk add --virtual builddeps \
       build-base \
       sqlite-dev \
    && bundle install --no-cache \
    && apk del builddeps \
    && rm -rf /var/cache/apk/*
RUN adduser -D massikone
RUN chown -R massikone:massikone .
RUN chown -R massikone:massikone /massikone/data
USER massikone
CMD puma -p 5000
