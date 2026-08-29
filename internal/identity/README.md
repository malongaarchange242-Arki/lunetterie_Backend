# Module Identity

Ce module couvre l'authentification, les sessions et la gestion des identités.

Règles de frontière:
- il ne doit pas manipuler directement le stock ni les commandes de réception ;
- il expose les services d'authentification et de session à travers des ports clairs ;
- l'authentification des utilisateurs reste indépendante du module Inventory.
