Steps to execute below test cases for LoM integration tests:

1. Navigate to this folder lom/src/plugins/sonic/plugin_integration_tests/linkcrc_mocker  
2. Execute "go build ."
3. Copy the generated binary linkcrc_mocker to star lab switch.
4. Execute the binary with below TestId as first arugment followed by space seperated list of interfaces. For ex, ./linkcrc_mocker 1 Ethernet96 Ethernet100



+--------+-----------------------------------+-----------+----------------------------------------------+----------------------+---------------------+
| TestId | Description                       | Detection | Max time out for integration test in seconds | Mock run for period  | Pattern of outliers |
+--------+-----------------------------------+-----------+----------------------------------------------+----------------------+---------------------+
| 0      | Continuous Outliers               | True      | 60 for first detection                       | 10 minutes           | 1                   |
+--------+-----------------------------------+-----------+----------------------------------------------+----------------------+---------------------+
| 1      | No Outliers                       | False     | No detection at any time                     | 10 minutes           | 0                   |
+--------+-----------------------------------+-----------+----------------------------------------------+----------------------+---------------------+
| 2      | Only first and last outlier       | True      | 240 for first detection                      | 10 minutes           | 1,0,0,0             |
+--------+-----------------------------------+-----------+----------------------------------------------+----------------------+---------------------+
| 3      | Only one outliers in 5 iterations | False     | No detection at any time                     | 10 minutes           | 1,0,0,0,0           |
+--------+-----------------------------------+-----------+----------------------------------------------+----------------------+---------------------+
