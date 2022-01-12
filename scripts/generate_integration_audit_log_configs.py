import os
import requests
import json

def get_audit_log_manifests(host, api_key):
    if not host or not api_key:
        raise Exception('host or api key not set')
    path_get_manifests = '/api/v2/integration-manifests'
    resp = requests.get(host + path_get_manifests, headers={'Authorization': api_key})
    if resp.status_code != 200:
        raise Exception(resp.status_code, 'unsuccessful get manifests request') 
    return filter_manifests(resp.json()['items'])

def filter_manifests(manifests):
    filtered = []
    for m in manifests:
        if 'capabilities' in m and 'auditLogEventsHook' in m['capabilities']:
            filtered.append(m)
    return filtered

def construct_config(manifest):
    """ takes an audit log manifest and returns the form variables in the format
    { <key>: {
        'type': <string>,
        'isOptional': <bool>,
        'allowedValues': <list>,
        'defaultValue': <interface>,
        'isSecret': <bool>
    } }
     """
    rawFormVariables = manifest['formVariables']
    formVariables = {}
    for rawV in rawFormVariables:
        v = { 'type': rawV['type'] }
        for attribute in ['isOptional', 'allowedValues', 'defaultValue', 'isSecret']:
            if attribute in rawV:
                v[attribute] = rawV[attribute]
        formVariables[rawV['key']] = v
    return formVariables

def construct_config_dict(manifests):
    cfgs = {}
    for m in manifests:
        cfgs[m['key']] = construct_config(m)
    return cfgs

def seed_config_file():
    host = os.getenv('LAUNCHDARKLY_API_HOST', 'https://app.launchdarkly.com')
    if not host.startswith('http'):
        host = 'https://' + host
    api_key = os.getenv('LAUNCHDARKLY_ACCESS_TOKEN')
    print('getting manifests...')
    manifests = get_audit_log_manifests(host, api_key)
    print('constructing configs...')
    configs = construct_config_dict(manifests)
    print('seeding file...')
    with open('launchdarkly/audit_log_subscription_configs.json', 'w') as f:
        json.dump(configs, f)
    print('COMPLETE, config data written to launchdarkly/audit_log_subscription_configs.json')

if __name__ == '__main__':
    seed_config_file()