FROM ruby:2.2.7-alpine

VOLUME /data
EXPOSE 5000

ENV BUNDLE_WITHOUT=mysql:pg
ENV RACK_ENV=deployment
ENV DATABASE_URL=sqlite:///data/massikone.db

RUN apk add --update imagemagick build-base sqlite sqlite-dev
RUN adduser -D massikone

COPY . /massikone
WORKDIR /massikone
RUN bundle install
RUN chown -R massikone:massikone .
RUN chown -R massikone:massikone /data
USER massikone
CMD puma -p 5000
