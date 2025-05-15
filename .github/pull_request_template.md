**Wait!**

- Have you added comprehensive tests?
- Have you updated relevant data sources as well as resources?
- Have you updated the docs?

**Testing**

For any changes you make, please ensure your acceptance test conform to the following:

- every single new attribute is tested
- optional attributes revert to null or default values if removed
- attributes that interact interact as expected
- block values behave as expected when reordered
- nested fields on maps or list/set items function as expected. Terraform does not actually enforce most schema attributes on nested items
- each test step for a configuration is followed by a test step where `ImportState` and `ImportStateVerify` are set to true. `ImportStateVerifyIgnore` can be used if we explicitly _expect_ a value to be different when imported, such as in the case of obfuscated values like API keys

## LaunchDarkly Employees
For more information on how to build, test, and release, see the [internal provider runbook](https://launchdarkly.atlassian.net/wiki/spaces/PD/pages/3825598468/LaunchDarkly+Terraform+Provider+Runbook).
