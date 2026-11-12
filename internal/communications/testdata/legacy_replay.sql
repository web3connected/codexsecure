-- Disposable bootstrap fixture, deliberately created before request binding exists.
INSERT INTO public.sites(site_key,name,base_url) VALUES ('go-legacy-replay-fixture','Legacy replay fixture','https://example.test');
SET ROLE codex_zones_runtime;
INSERT INTO public.communication_objects(site_id,registration_key,object_type,detected_url,detected_method,detected_zone,canonical_destination,scope)
SELECT id,'00000000-0000-4000-8000-000000000071','fetch','https://example.test/a','GET','Z1','https://example.test/a','universal' FROM public.sites WHERE site_key='go-legacy-replay-fixture';
SELECT public.codex_communications_save_draft('go-legacy-replay-fixture','00000000-0000-4000-8000-000000000071',
 '{"schema_version":1,"destination":"/a","destination_match":"exact","methods":["GET"],"type":"fetch","zone":"Z1","enabled":true}',
 0,'admin-1','00000000-0000-4000-8000-000000000072');
RESET ROLE;
