FROM ruby:2.2.7-alpine
RUN apk add --update imagemagick build-base sqlite-dev
RUN echo "gem: --no-rdoc --no-ri" > /etc/gemrc
RUN addgroup massikone && adduser -D -h /massikone -G massikone massikone
ENV BUNDLE_WITHOUT=mysql:pg
ENV RACK_ENV=deployment
ENV DATABASE_URL=sqlite:///data/massikone.sqlite
RUN mkdir /data && chown massikone:massikone /data
EXPOSE 5000
VOLUME /data
WORKDIR /massikone
COPY . /massikone/
RUN chown -R massikone:massikone /massikone
RUN (cd /massikone && bundle install)
USER massikone
CMD puma -p 5000