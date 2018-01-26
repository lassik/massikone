/* Tulosteen sisältö on tallennettu CSV-tiedostoon, jossa
 * erottimena käytetään ;-merkkiä. Ensimmäinen kenttä sisältää
 * kolme merkkiä:
 *
 * 1. merkki 'D': Tilierittelyt.
 * 1. merkki 'H': Tulostetaan otsikkorivi, ei rahamäärää.
 *                Rivi näytetään aina.
 * 1. merkki 'G': Tulostetaan otsikkorivi, ei rahamäärää. Rivi
 *                näytetään vain, jos summa on erisuuri kuin 0,00.
 * 1. merkki 'S': Tulostetaan teksti ja rahamäärä.
 *                Rivi näytetään aina.
 * 1. merkki 'T': Tulostetaan teksti ja rahamäärä. Rivi
 *                näytetään vain, jos summa on erisuuri kuin 0,00.
 * 2. merkki 'P': Tekstiä ei lihavoida eikä kursivoida.
 * 2. merkki 'B': Teksti lihavoidaan.
 * 2. merkki 'I': Teksti kursivoidaan.
 * 3. merkki:     Ilmoittaa, kuinka paljon tekstiä sisennetään.
 *                Kokonaisluku välillä 0 .. 9.
 *
 * Seuraavat kentät ilmoittavat, miltä tilinumeroväleiltä
 * summa lasketaan. Kenttiä on oltava parillinen määrä,
 * jokaisen välin alku- ja loppunumero.
 *
 * Viimeinen kenttä on tulostettava teksti.
 */
