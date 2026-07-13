-- Seed data for epure-shop
-- Admin password: admin123

INSERT OR IGNORE INTO users (id, email, name, password_hash, role, created_at) VALUES (
    '550e8400-e29b-41d4-a716-446655440001',
    'admin@epure.com',
    'Admin',
    '$2a$12$rGi2f4YbeHPvr./fITbcvuhNagmCW2x3AoPQK6cCVgji8aOZ7w.iO',
    'admin',
    '2025-01-01T00:00:00Z'
);

-- Categories (8)
INSERT OR IGNORE INTO categories (id, slug, name, parent_id) VALUES
    ('c001-0000-0000-000000000001', 'parfums',    'Parfums',    NULL),
    ('c001-0000-0000-000000000002', 'soins',      'Soins',      NULL),
    ('c001-0000-0000-000000000003', 'maquillage', 'Maquillage', NULL),
    ('c001-0000-0000-000000000004', 'cheveux',    'Cheveux',    NULL),
    ('c001-0000-0000-000000000005', 'corps',      'Corps',      NULL),
    ('c001-0000-0000-000000000006', 'hommes',     'Hommes',     NULL),
    ('c001-0000-0000-000000000007', 'accessoires','Accessoires',NULL),
    ('c001-0000-0000-000000000008', 'coffrets',   'Coffrets',   NULL);

-- Products (40, 5 per category)
INSERT OR IGNORE INTO products (id, slug, category_id, name, description, price, compare_price, stock, status, image_url, rating, review_count, created_at) VALUES
-- Parfums (c001-0000-0000-000000000001)
('p001-0000-0000-000000000001', 'eau-de-rose',         'c001-0000-0000-000000000001', 'Eau de Rose',         'Un parfum floral délicat aux notes de rose de Damas.',              8900,  9900,  50, 'published', '/static/img/placeholder.svg', 4.5, 120, '2025-01-01T00:00:00Z'),
('p001-0000-0000-000000000002', 'ambre-mystique',      'c001-0000-0000-000000000001', 'Ambre Mystique',      'Un parfum envoûtant mêlant ambre, vanille et bois de santal.',      12900, NULL,  35, 'published', '/static/img/placeholder.svg', 4.8, 89,  '2025-01-02T00:00:00Z'),
('p001-0000-0000-000000000003', 'citron-vert',         'c001-0000-0000-000000000001', 'Citron Vert',         'Une eau de toilette fraîche et pétillante aux agrumes.',             5400,  NULL,  100,'published', '/static/img/placeholder.svg', 4.2, 200, '2025-01-03T00:00:00Z'),
('p001-0000-0000-000000000004', 'jasmin-nuit',         'c001-0000-0000-000000000001', 'Jasmin Nuit',         'Un parfum sensuel pour les soirées, jasmin et musc blanc.',          11000, 12900, 20, 'published', '/static/img/placeholder.svg', 4.6, 67,  '2025-01-04T00:00:00Z'),
('p001-0000-0000-000000000005', 'bois-de-cedre',       'c001-0000-0000-000000000001', 'Bois de Cèdre',       'Un parfum boisé et masculin, cèdre et notes épicées.',              7500,  NULL,  45, 'published', '/static/img/placeholder.svg', 4.3, 150, '2025-01-05T00:00:00Z'),

-- Soins (c001-0000-0000-000000000002)
('p001-0000-0000-000000000006', 'creme-hydratante',    'c001-0000-0000-000000000002', 'Crème Hydratante',    'Crème visage hydratante à l''acide hyaluronique.',                    3400,  4200,  80, 'published', '/static/img/placeholder.svg', 4.7, 310, '2025-01-06T00:00:00Z'),
('p001-0000-0000-000000000007', 'serum-eclat',         'c001-0000-0000-000000000002', 'Sérum Éclat',         'Sérum concentré à la vitamine C pour un teint lumineux.',            4900,  NULL,  60, 'published', '/static/img/placeholder.svg', 4.9, 178, '2025-01-07T00:00:00Z'),
('p001-0000-0000-000000000008', 'contour-des-yeux',    'c001-0000-0000-000000000002', 'Contour des Yeux',    'Soin contour des yeux anti-poches et anti-cernes.',                   2800,  NULL,  90, 'published', '/static/img/placeholder.svg', 4.4, 95,  '2025-01-08T00:00:00Z'),
('p001-0000-0000-000000000009', 'masque-argile',       'c001-0000-0000-000000000002', 'Masque à l''Argile',  'Masque purifiant à l''argile verte pour peaux mixtes.',              1900,  2400,  120,'published', '/static/img/placeholder.svg', 4.1, 230, '2025-01-09T00:00:00Z'),
('p001-0000-0000-000000000010', 'huile-demaquillante', 'c001-0000-0000-000000000002', 'Huile Démaquillante', 'Huile démaquillante douce qui élimine même le maquillage waterproof.',2200, NULL,  70, 'published', '/static/img/placeholder.svg', 4.6, 145, '2025-01-10T00:00:00Z'),

-- Maquillage (c001-0000-0000-000000000003)
('p001-0000-0000-000000000011', 'fond-de-teint',       'c001-0000-0000-000000000003', 'Fond de Teint Fluide','Fond de teint léger et modulable, fini naturel.',                    3800,  NULL,  55, 'published', '/static/img/placeholder.svg', 4.3, 189, '2025-01-11T00:00:00Z'),
('p001-0000-0000-000000000012', 'rouge-a-levres',      'c001-0000-0000-000000000003', 'Rouge à Lèvres Mat', 'Rouge à lèvres longue tenue, fini mat velouté.',                     2400,  2900,  100,'published', '/static/img/placeholder.svg', 4.5, 320, '2025-01-12T00:00:00Z'),
('p001-0000-0000-000000000013', 'palette-ombres',      'c001-0000-0000-000000000003', 'Palette d''Ombres',  'Palette de 12 ombres à paupières, finis mats et irisés.',            4500,  NULL,  40, 'published', '/static/img/placeholder.svg', 4.7, 210, '2025-01-13T00:00:00Z'),
('p001-0000-0000-000000000014', 'mascara-volume',      'c001-0000-0000-000000000003', 'Mascara Volume',      'Mascara effet volume et courbe, sans paquets.',                       1900,  NULL,  130,'published', '/static/img/placeholder.svg', 4.2, 400, '2025-01-14T00:00:00Z'),
('p001-0000-0000-000000000015', 'blush-peche',         'c001-0000-0000-000000000003', 'Blush Pêche',         'Blush poudre teinte pêche, effet bonne mine.',                        2200,  NULL,  75, 'published', '/static/img/placeholder.svg', 4.4, 156, '2025-01-15T00:00:00Z'),

-- Cheveux (c001-0000-0000-000000000004)
('p001-0000-0000-000000000016', 'shampoing-reparateur','c001-0000-0000-000000000004', 'Shampoing Réparateur','Shampoing sans sulfates qui répare les cheveux abîmés.',            1800,  2200,  150,'published', '/static/img/placeholder.svg', 4.6, 280, '2025-01-16T00:00:00Z'),
('p001-0000-0000-000000000017', 'apres-shampoing',     'c001-0000-0000-000000000004', 'Après-Shampoing',     'Après-shampoing démêlant à l''huile d''argan.',                       1600,  NULL,  120,'published', '/static/img/placeholder.svg', 4.3, 195, '2025-01-17T00:00:00Z'),
('p001-0000-0000-000000000018', 'masque-cheveux',      'c001-0000-0000-000000000004', 'Masque Cheveux',      'Masque capillaire nourrissant au beurre de karité.',                  2100,  NULL,  65, 'published', '/static/img/placeholder.svg', 4.5, 134, '2025-01-18T00:00:00Z'),
('p001-0000-0000-000000000019', 'huile-cheveux',       'c001-0000-0000-000000000004', 'Huile Cheveux',       'Huile sèche sublimante pour cheveux brillants.',                      2500,  3000,  50, 'published', '/static/img/placeholder.svg', 4.1, 88,  '2025-01-19T00:00:00Z'),
('p001-0000-0000-000000000020', 'spray-thermoprotecteur','c001-0000-0000-000000000004','Spray Thermoprotecteur','Protection thermique avant brushing et lissage.',             1400,  NULL,  85, 'published', '/static/img/placeholder.svg', 4.4, 167, '2025-01-20T00:00:00Z'),

-- Corps (c001-0000-0000-000000000005)
('p001-0000-0000-000000000021', 'lait-corps',          'c001-0000-0000-000000000005', 'Lait Corps Hydratant','Lait corporel à la texture fondante, hydratation 24h.',           1500,  NULL,  200,'published', '/static/img/placeholder.svg', 4.5, 350, '2025-01-21T00:00:00Z'),
('p001-0000-0000-000000000022', 'gommage-corps',       'c001-0000-0000-000000000005', 'Gommage Corps',       'Gommage exfoliant aux grains de sucre et huile d''amande.',         2200,  NULL,  60, 'published', '/static/img/placeholder.svg', 4.7, 178, '2025-01-22T00:00:00Z'),
('p001-0000-0000-000000000023', 'beurre-corps',        'c001-0000-0000-000000000005', 'Beurre Corporel',     'Beurre corporel ultra-nourrissant au beurre de karité.',            1800,  2100,  45, 'published', '/static/img/placeholder.svg', 4.8, 220, '2025-01-23T00:00:00Z'),
('p001-0000-0000-000000000024', 'deodorant-doux',      'c001-0000-0000-000000000005', 'Déodorant Doux',      'Déodorant sans aluminium, protection 48h.',                           900,  NULL,  300,'published', '/static/img/placeholder.svg', 4.0, 410, '2025-01-24T00:00:00Z'),
('p001-0000-0000-000000000025', 'huile-massage',       'c001-0000-0000-000000000005', 'Huile de Massage',    'Huile de massage relaxante aux huiles essentielles.',                1900,  NULL,  55, 'published', '/static/img/placeholder.svg', 4.6, 132, '2025-01-25T00:00:00Z'),

-- Hommes (c001-0000-0000-000000000006)
('p001-0000-0000-000000000026', 'baume-apres-rasage',  'c001-0000-0000-000000000006', 'Baume Après-Rasage',  'Baume apaisant sans alcool, parfum boisé discret.',                   1600,  NULL,  70, 'published', '/static/img/placeholder.svg', 4.3, 95,  '2025-01-26T00:00:00Z'),
('p001-0000-0000-000000000027', 'eau-de-toilette-homme','c001-0000-0000-000000000006','Eau de Toilette Homme','Eau de toilette fraîche aux notes de vétiver et bergamote.',     5900,  NULL,  30, 'published', '/static/img/placeholder.svg', 4.5, 145, '2025-01-27T00:00:00Z'),
('p001-0000-0000-000000000028', 'gel-douche-homme',    'c001-0000-0000-000000000006', 'Gel Douche Homme',    'Gel douche tonifiant au charbon actif.',                              1200,  NULL,  110,'published', '/static/img/placeholder.svg', 4.1, 267, '2025-01-28T00:00:00Z'),
('p001-0000-0000-000000000029', 'creme-rasage',        'c001-0000-0000-000000000006', 'Crème à Raser',       'Crème à raser onctueuse qui protège les peaux sensibles.',           1400,  NULL,  90, 'published', '/static/img/placeholder.svg', 4.4, 78,  '2025-01-29T00:00:00Z'),
('p001-0000-0000-000000000030', 'coffret-barbe',       'c001-0000-0000-000000000006', 'Coffret Barbe',       'Huile, baume et peigne pour barbe parfaitement entretenue.',          3500,  4200,  25, 'published', '/static/img/placeholder.svg', 4.7, 56,  '2025-01-30T00:00:00Z'),

-- Accessoires (c001-0000-0000-000000000007)
('p001-0000-0000-000000000031', 'pinceau-fond-de-teint','c001-0000-0000-000000000007','Pinceau Fond de Teint','Pinceau kabuki haute densité pour un fini zéro trace.',          1500,  NULL,  80, 'published', '/static/img/placeholder.svg', 4.2, 112, '2025-01-31T00:00:00Z'),
('p001-0000-0000-000000000032', 'eponge-beaute',       'c001-0000-0000-000000000007', 'Éponge Beauté',       'Éponge à maquillage extra-douce, application homogène.',              800,  NULL,  200,'published', '/static/img/placeholder.svg', 4.6, 340, '2025-02-01T00:00:00Z'),
('p001-0000-0000-000000000033', 'trousse-maquillage',  'c001-0000-0000-000000000007', 'Trousse Maquillage',  'Trousse de rangement en tissu lavable, 4 compartiments.',             1800,  NULL,  40, 'published', '/static/img/placeholder.svg', 4.3, 67,  '2025-02-02T00:00:00Z'),
('p001-0000-0000-000000000034', 'miroir-grossissant',  'c001-0000-0000-000000000007', 'Miroir Grossissant',  'Miroir de précision x10 avec ventouse.',                               1200, 1500,  65, 'published', '/static/img/placeholder.svg', 4.0, 89,  '2025-02-03T00:00:00Z'),
('p001-0000-0000-000000000035', 'brosse-cheveux',      'c001-0000-0000-000000000007', 'Brosse en Bambou',    'Brosse démêlante en bambou écologique.',                              1100,  NULL,  100,'published', '/static/img/placeholder.svg', 4.5, 210, '2025-02-04T00:00:00Z'),

-- Coffrets (c001-0000-0000-000000000008)
('p001-0000-0000-000000000036', 'coffret-decouverte',  'c001-0000-0000-000000000008', 'Coffret Découverte',  'Miniatures de nos 5 parfums iconiques.',                               3900,  4900,  30, 'published', '/static/img/placeholder.svg', 4.9, 450, '2025-02-05T00:00:00Z'),
('p001-0000-0000-000000000037', 'coffret-soin-visage', 'c001-0000-0000-000000000008', 'Coffret Soin Visage', 'Routine complète : nettoyant, sérum et crème hydratante.',            6900,  NULL,  20, 'published', '/static/img/placeholder.svg', 4.8, 178, '2025-02-06T00:00:00Z'),
('p001-0000-0000-000000000038', 'coffret-bain',        'c001-0000-0000-000000000008', 'Coffret Bain Relax',  'Sels de bain, bougie parfumée et huile de massage.',                  4500,  NULL,  35, 'published', '/static/img/placeholder.svg', 4.6, 95,  '2025-02-07T00:00:00Z'),
('p001-0000-0000-000000000039', 'coffret-homme',       'c001-0000-0000-000000000008', 'Coffret Homme',       'Eau de toilette, gel douche et baume après-rasage.',                  5500,  6500,  15, 'published', '/static/img/placeholder.svg', 4.4, 67,  '2025-02-08T00:00:00Z'),
('p001-0000-0000-000000000040', 'calendrier-avent',    'c001-0000-0000-000000000008', 'Calendrier de l''Avent','24 surprises beauté pour patienter jusqu''à Noël.',             8900,  NULL,  10, 'published', '/static/img/placeholder.svg', 4.9, 512, '2025-02-09T00:00:00Z');

-- Discount
INSERT OR IGNORE INTO discounts (id, code, percent, active, expires_at) VALUES (
    'd001-0000-0000-000000000001',
    'WELCOME10',
    10,
    1,
    '2027-12-31T23:59:59Z'
);
