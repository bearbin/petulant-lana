petulant-lana
=============

The prototype for a file sharing website I'm working on.

Use with caution, there are still some issues I'm working on.

MIT Licensed.

Donate: 19DjXPi3SRPLHizWvXaNN9DuxQnCvgw5qj

Setup
-----

 1. In `config.json`, edit the settings:
   * Set `name` to the name you want for your host.
   * Set `url` to the url for your host.
   * Set `callbacksecret` to a secret callback endpoint. (`hagh243akghjkahg67q5894eyhauhgakjh4234fakj` is a good example. `url/callbacksecret` is the url for the callback to put into coinbase.)
   * Set `baseprice` to the base price per megabyte (in satoshis).
   * Set `minprice` to the minimum price per file (in satoshis).
   * Set `coinbasekey` to your Coinbase API key (for billing customers).
 2. Start Petulant-Lana. Run in su mode if you want port 80, otherwise 8080 will be selected..
