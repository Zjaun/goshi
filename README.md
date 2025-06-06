# Go Operating System and Hardware Information
This is a stripped-down port of the OSHI library in Java. This repo will be mainly used as a dependency for another port of a Java application.
Mostly follows the patterns used in the original project.

Currently, it only provides basic information for the processor, memory, and graphics card.

If you want to contribute, feel free to do so in any means! Fork, pull, however you like!

# Prerequisites
When used in a Windows application:
* It must have an application manifest. This is because the Windows version needs to be determined
for it to work properly, like querying the `Win32_PhysicalMemory` class. In Windows 10 and above, it has the `SMBiosMemoryType` field.
Whereas in versions lower than that, it only has the `MemoryType` field.
