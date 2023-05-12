Link CRC Detection Algorithm

For every 30 seconds:
- Collect "SAI_PORT_STAT_IF_IN_ERRORS", "SAI_PORT_STAT_IF_OUT_ERRORS", "SAI_PORT_STAT_IF_IN_UCAST_PKTS", "SAI_PORT_STAT_IF_OUT_UCAST_PKTS" counters for 
  Ethernet and Management interfaces which have both admin_status and oper_status as up.
- Proceed if below conditions are true (here diff represents difference of current counter w.r.t to previous counter) :
   a) Validate if all new counters are greater than previous counters.
   b) IfInErrorsDiff >0 and ((InUnicastPacketsDiff > 100) or (OutUnicastPacketsDiff > 100)
   c) (IfInErrorsDiff / (InUnicastPacketsDiff + IfInErrorsDiff)) > 0.000001
- In a sliding window of 125 seconds (i.e, 5 most recent data points), if below is true for more than two data points, crc is detected.
   a) ((IfInErrorsDiff - IfOutErrorsDiff)/InUnicastPacketsDiff) > 0
   
   
References:
- https://msazure.visualstudio.com/One/_git/Networking-Phynet-DeviceHealthAlerting?path=/src/ASA/ASAJobs/LinkLayerCRC/Script.asaql
   
   
