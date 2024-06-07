[![CC BY-NC-SA 4.0][cc-by-nc-sa-shield]][cc-by-nc-sa]

[cc-by-nc-sa]: http://creativecommons.org/licenses/by-nc-sa/4.0/
[cc-by-nc-sa-shield]: https://img.shields.io/badge/License-CC%20BY--NC--SA%204.0-lightgrey.svg

# Bank Ninja Chat System

### Local Sentry Deploy

```shell
COMPOSE_PROFILES=sentry task deps
task deps:cmd -- exec sentry sentry upgrade
task deps:cmd -- restart sentry
```

If you want to recreate a user:

```shell
task deps:cmd -- run --rm sentry createuser  
```