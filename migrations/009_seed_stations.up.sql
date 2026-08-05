-- Sous-stations utilisées par le flux d'envoi/réception (scan.html)
INSERT INTO stations (name, type, city)
SELECT 'Station Pointe-Noire', 'SOUS_STATION', 'Pointe-Noire'
WHERE NOT EXISTS (SELECT 1 FROM stations WHERE name = 'Station Pointe-Noire');

INSERT INTO stations (name, type, city)
SELECT 'Présentoir', 'SOUS_STATION', 'Brazzaville'
WHERE NOT EXISTS (SELECT 1 FROM stations WHERE name = 'Présentoir');

INSERT INTO stations (name, type, city)
SELECT 'Laboratoire', 'SOUS_STATION', 'Brazzaville'
WHERE NOT EXISTS (SELECT 1 FROM stations WHERE name = 'Laboratoire');
