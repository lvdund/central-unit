
Files: **3GPP TS 38.331 (Release 17)**.

---

## 1️⃣ RRC Message Containers (outer wrappers)

These are the **top-level ASN.1 message types** — each defines the context and direction of transmission.
Every RRC message *must* be encoded inside one of them.

| Direction     | Channel Type       | ASN.1 Container         | Description / Purpose                                                                         |
| ------------- | ------------------ | ----------------------- | --------------------------------------------------------------------------------------------- |
| **Downlink**  | Common control     | **DL-CCCH-Message**     | Used before UE has a dedicated RRC connection (e.g. `RRCSetup`).                              |
| **Downlink**  | Dedicated control  | **DL-DCCH-Message**     | Used after UE has an RRC connection (e.g. `RRCReconfiguration`, `SecurityModeCommand`).       |
| **Uplink**    | Common control     | **UL-CCCH-Message**     | Used before RRC connection exists (e.g. `RRCSetupRequest`, `RRCReestablishmentRequest`).      |
| **Uplink**    | Dedicated control  | **UL-DCCH-Message**     | Used after RRC connection is set up (e.g. `RRCReconfigurationComplete`, `MeasurementReport`). |
| **Broadcast** | System information | **BCCH-DL-SCH-Message** | Carries `SystemInformationBlockType1` (SIB1).                                                 |
| **Broadcast** | Paging             | **PCCH-Message**        | Carries paging records (`PagingRecordList`).                                                  |
| **Broadcast** | Master information | **BCCH-BCH-Message**    | Carries `MIB` (Master Information Block).                                                     |

So the full set of outer RRC “frames” is:

> **{ DL-CCCH, DL-DCCH, UL-CCCH, UL-DCCH, BCCH-BCH, BCCH-DL-SCH, PCCH }**

---

## 2️⃣ RRC Message Types (the actual messages)

Now, inside those containers, you find the **actual RRC messages**.
Here’s the full inventory from TS 38.331 §6.2 (grouped by purpose):

### a. **Connection establishment and maintenance**

| Message                      | Direction | Container |
| ---------------------------- | --------- | --------- |
| `RRCSetupRequest`            | UL        | UL-CCCH   |
| `RRCSetup`                   | DL        | DL-CCCH   |
| `RRCSetupComplete`           | UL        | UL-DCCH   |
| `RRCReconfiguration`         | DL        | DL-DCCH   |
| `RRCReconfigurationComplete` | UL        | UL-DCCH   |
| `RRCRelease`                 | DL        | DL-DCCH   |
| `RRCResumeRequest`           | UL        | UL-CCCH   |
| `RRCResume`                  | DL        | DL-CCCH   |
| `RRCResumeComplete`          | UL        | UL-DCCH   |
| `RRCReestablishmentRequest`  | UL        | UL-CCCH   |
| `RRCReestablishment`         | DL        | DL-CCCH   |
| `RRCReestablishmentComplete` | UL        | UL-DCCH   |
| `RRCReject`                  | DL        | DL-CCCH   |

### b. **Security and mobility**

| Message                                     | Direction | Container |
| ------------------------------------------- | --------- | --------- |
| `SecurityModeCommand`                       | DL        | DL-DCCH   |
| `SecurityModeComplete`                      | UL        | UL-DCCH   |
| `SecurityModeFailure`                       | UL        | UL-DCCH   |
| `HandoverPreparationInformation`            | DL        | DL-DCCH   |
| `HandoverCommand`                           | DL        | DL-DCCH   |
| `HandoverPreparationInformationAcknowledge` | UL        | UL-DCCH   |
| `HandoverFailure`                           | UL        | UL-DCCH   |

### c. **Measurement and configuration**

| Message                         | Direction | Container |
| ------------------------------- | --------- | --------- |
| `MeasurementReport`             | UL        | UL-DCCH   |
| `MeasurementReportAppLayer`     | UL        | UL-DCCH   |
| `MeasurementReportAppLayerConf` | DL        | DL-DCCH   |
| `CounterCheck`                  | DL        | DL-DCCH   |
| `CounterCheckResponse`          | UL        | UL-DCCH   |

### d. **System information and paging**

| Message                                  | Direction | Container   |
| ---------------------------------------- | --------- | ----------- |
| `MIB`                                    | DL        | BCCH-BCH    |
| `SystemInformationBlockType1`            | DL        | BCCH-DL-SCH |
| `SystemInformation` (SIB2 … SIB24, etc.) | DL        | BCCH-DL-SCH |
| `Paging`                                 | DL        | PCCH        |
| `RRCSystemInformationRequest`            | UL        | UL-DCCH     |

### e. **Miscellaneous / advanced procedures**

| Message                                          | Direction | Container         |
| ------------------------------------------------ | --------- | ----------------- |
| `UEInformationRequest` / `UEInformationResponse` | DL/UL     | DL-DCCH / UL-DCCH |
| `UEAssistanceInformation`                        | UL        | UL-DCCH           |
| `RRCReestablishmentReject`                       | DL        | DL-CCCH           |
| `RRCReleaseComplete`                             | UL        | UL-DCCH           |

---

### ✅ So, to summarize:

**Outer RRC message containers:**

> `DL-CCCH`, `DL-DCCH`, `UL-CCCH`, `UL-DCCH`, `BCCH-BCH`, `BCCH-DL-SCH`, `PCCH`

**Actual RRC messages (examples):**

> `RRCSetupRequest`, `RRCSetup`, `RRCSetupComplete`, `RRCReconfiguration`, `RRCRelease`,
> `RRCResume`, `RRCReestablishment`, `SecurityModeCommand`, `MeasurementReport`,
> `SystemInformationBlockType1`, `Paging`, etc.

---


