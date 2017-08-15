# Massikone

Yhdistyksen kirjanpito netissä

Demo: https://massikone.herokuapp.com/

Valmis Docker-kontti: https://hub.docker.com/r/lassik/massikone/

Lähdekoodi: https://github.com/lassik/massikone

# Kehittäminen

1. Rekisteröi sovellus Google Cloud-hallintapaneelissa. Lisää *Authorized redirect URIs* osoite `http://127.0.0.1:3000/auth/google_oauth2/callback`
2. Luo `.env`-niminen tiedosto käyttäen mallina `.env.sample.docker`
3. Käynnistä palvelinohjelma Docker-konttiin: `docker-compose up`
4. Mene selaimella osoitteeseen `http://127.0.0.1:3000/`
5. `public/static` ja `views`-hakemistojen muokkaukset näkyvät suoraan. Ruby-koodia muokattuasi sulje palvelin (*Ctrl+C*) ja käynnistä se uudelleen.
