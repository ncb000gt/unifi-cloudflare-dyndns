Unifi -> Cloudflare DYNDNS
=

I wanted to update a Cloudflare DNS record with my external gateway IP through my Unifi Controller. This does that.


Info
==

You'll need a config file. I used `config.json` although [Viper](https://github.com/spf13/viper) supports more than that, so you can check that out there.

The tool expects a config structure like:

    {
      "unifi": {
        "host": "",
        "username": "",
        "password": ""
      },
      "cloudflare": {
        "authEmail": "",
        "authKey": "",
        "dnsName": "",
        "zoneName": ""
      }
    }

Most of this should be straight forward, but `dnsName` should match the record you're trying to update. `zoneName` should match the zone you're trying to work inside of.


Other
==

I wrote this in a couple hours so I'm sure this isn't 100%. If you find issues, let me know and I may have time to look into them.
