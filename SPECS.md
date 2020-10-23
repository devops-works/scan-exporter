# Scan exporter

## Objectif

Le but de ce projet est de remplacer les scans nmap réalisés via Jenkins, et de
collecter les résultats via prometheus pour la métrologie.

Un fichier de configuration écrit en YAML comportera les différentes cibles qui
seront scannées. Ce fichier sera lu et interprété par un programme en Go qui
fera les scans et exposera les métriques.

## Fichier de configuration

Example :

```yaml
targets:
  - name: "app.db"
    period: "1d"
    ip: "x.x.x.x"
    tcp:
      range: "all"
      expected: "22,33,44"
    udp:
      range: "1-1024"
      expected: "22,53"

  - name: "app.web"
    period: "6h"
    ip: "y.y.y.y"
    tcp:
      range: "all"
      expected: "22,33,44"
    udp:
      range: "1-1024"
      expected: "22,53"
```

* `targets` contient la liste des cibles qui vont être scannées.
* `name` est le nom de l'application scannée.
* `period` représente la fréquence à laquelle le scan des ports sera effectué.
  Les unités de temps peuvent être:
  * `d` pour les jours ;
  * `h` pour les heures ;
  * `m` pour les minutes.
* `ip` est le nom de l'application scannée.
* `tcp` et `udp` peuvent chacun contenir des ranges différents, c'est pour ça
  qu'ils sont séparés en deux aprties distinctes
* `range` est une string qui indique quels ports scanner. Les options possibles
  sont :
  * `all` : ports 1 à 65535 ;
  * `reserved` : ports 1 à 1024 ;
  * `x-y` : ports x à y ;
  * `x,y` : ports x et y ;
  * `w-x,y-z` : ports w à x, et ports y à z.