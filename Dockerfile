FROM ruby:2.2.7-alpine

VOLUME /massikone/data
EXPOSE 5000

ENV BUNDLE_WITHOUT=mysql:pg
ENV RACK_ENV=deployment

COPY . /massikone
WORKDIR /massikone
RUN apk add --update imagemagick build-base sqlite sqlite-libs sqlite-dev
RUN adduser -D massikone
RUN bundle install --no-cache
RUN chown -R massikone:massikone .
RUN chown -R massikone:massikone /massikone/data
USER massikone
CMD puma -p 5000
