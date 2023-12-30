
<h1 align="center">
Swim - CertStream Data Processor
</h1>

<p align="center">
  <img src="resources/logo.png" alt="Swim Logo" width="400">
</p>

Swim is a Go application designed to interface with the CertStream service, processing incoming SSL/TLS certificate transparency logs and extracting domain data. This data is stored in a SQLite database for efficient access and analysis. 

Additionally, Swim offers a RESTful API, allowing users to query and retrieve processed domain information, including details about domain names, certificate update events, and subdomains. This makes Swim an invaluable tool for security analysts and researchers interested in real-time monitoring of certificate issuance and domain changes.

<h2 align="center">
Installation
</h2>

### Prerequisites
- Go (Version 1.x.x or higher)

### Using Go Install
If you have Go installed and configured (with `GOPATH` set up), you can directly install Swim using the `go install` command:

```bash
go install github.com/dap-ware/swim@latest
```

### Using git clone and building from source
Alternatively, you can clone the repository and build Swim manually. This is a good option if you want to work with the source code or contribute to the project:

1. Clone the repository:
   ```bash
   git clone https://github.com/dap-ware/swim.git
   cd swim
   ```
2. Build the application using Go:
   ```bash
   go build .
   ```
   This will compile the Swim application and create an executable file in the current directory.
   
<h2 align="center">
Usage
</h2>
Swim allows you to specify the SQLite database file and the batch size for processing using the `-db` and `-bs` flags, respectively. 

- Default database file: `swim.db`
- Default batch size: `1000`

### Running Swim
To run Swim with custom settings:
```bash
./swim -db your_database.db -bs 500
```
To display help information:
```bash
./swim -h
```
<h2 align="center">
API Reference
</h2>
<h3 align="center">
Fetch Domain Names (Apex Domains)
</h3>

---
**Endpoint**: `GET /v1/domains`
This endpoint retrieves all apex domain names stored in the database. It's particularly useful for getting a comprehensive list of top-level domains being monitored for certificate updates.

#### Query Parameters
- `page`: Page number for pagination (default: 1)
- `size`: Number of domain names per page (default: 1000)

#### Example Request
To fetch the first page of apex domain names with a limit of 10 domain names:
```bash
GET http://localhost:8080/v1/domains?page=1&size=10
```
#### Example Response
```json
[
  "panelpool.de",
  "prismaticasa.com",
  "shelftheelf.com",
  "visabitech.com",
  "isabolic.dev",
  "wxawnting-ornxawment.shop",
  "petsearchandrescueinc.org",
  "auto-umfrage.ch",
  "bancochile.bond",
  "kc-universities-with-fully-funded-doctoral-programs.today"
]
```
---
<h3 align="center">
Fetch Domain Specific Cert Update Event Data
</h3>

---
**Endpoint**: `GET /v1/get/domains`
- This endpoint retrieves certificate update event data for domains. The data includes details such as domain names, their apex status, parent domains, SSL certificate information, and more.

#### **Query Parameters**
- `page`: Page number for pagination (default: 1)
- `size`: Number of domain records per page (default: 1000)

#### **Example Request**
- To fetch the first page of domain event data with 2 records per page:
```bash
GET http://localhost:8080/v1/get/domains?page=1&size=2
```
#### **Example Response**
```json
[
  {
    "domain": "148558com-tz3.zhuzhana1.com",
    "is_apex": false,
    "parent_domain": "zhuzhana1.com",
    "not_before": "2023-12-30T10:10:42-05:00",
    "serial_number": "4E074B3B16ADB6C8272FA71204C5E10F3B5",
    "fingerprint": "AA:6D:18:B9:23:79:7D:D3:AE:18:8B:4D:ED:FB:11:7E:E7:67:53:7D",
    "key_usage": "Digital Signature, Key Encipherment",
    "extended_key_usage": "TLS Web server authentication, TLS Web client authentication",
    "subject_key_id": "7D:6D:F5:48:81:1C:F2:24:06:62:1E:36:E0:69:81:A1:FF:45:F8:26",
    "authority_key_id": "keyid:14:2E:B3:17:B7:58:56:CB:AE:50:09:40:E6:1F:AF:9D:8B:14:C2:C6\n",
    "authority_info": "CA Issuers - URI:http://r3.i.lencr.org/\nOCSP - URI:http://r3.o.lencr.org\n",
    "subject_alt_name": "DNS:148558com-tz3.zhuzhana1.com, DNS:148558com-tz2.zhuzhana1.com, DNS:148558com-tz1.zhuzhana1.com",
    "certificate_policies": "Policy: 2.23.140.1.2.1",
    "wildcard": false
  },
  {
    "domain": "14881337.xyz",
    "is_apex": true,
    "parent_domain": "",
    "not_before": "2023-12-30T10:20:34-05:00",
    "serial_number": "3C1F1B2638C0543F3707936C1F62052D9C7",
    "fingerprint": "11:24:2F:8D:13:53:A2:07:E3:2C:B6:B9:C2:7B:A2:65:46:DF:47:74",
    "key_usage": "Digital Signature, Key Encipherment",
    "extended_key_usage": "TLS Web server authentication, TLS Web client authentication",
    "subject_key_id": "34:F8:7F:A2:5C:C6:83:DD:91:EE:FF:8D:0D:A4:A1:FC:94:98:AF:D4",
    "authority_key_id": "keyid:14:2E:B3:17:B7:58:56:CB:AE:50:09:40:E6:1F:AF:9D:8B:14:C2:C6\n",
    "authority_info": "CA Issuers - URI:http://r3.i.lencr.org/\nOCSP - URI:http://r3.o.lencr.org\n",
    "subject_alt_name": "DNS:14881337.xyz, DNS:*.14881337.xyz",
    "certificate_policies": "Policy: 2.23.140.1.2.1",
    "wildcard": true
  }
]
```
---
<h3 align="center">
Fetch Subdomains
</h3>

---
**Endpoint**: `GET /v1/get/:domain/subdomains`
- This endpoint retrieves a list of subdomains for a given domain name. It is useful for identifying all subdomains associated with a specific apex domain, which can be crucial for domain management and security analysis.

#### **Path Parameters**
- `domain`: The domain name for which subdomains are to be fetched. For example, `dynamic-m.com`.

#### **Example Request**
To fetch subdomains for the domain `dynamic-m.com`:
```bash
GET http://localhost:8080/v1/get/dynamic-m.com/subdomains
```

#### **Example Response**
- The response is streamed as JSON arrays, here is the example response from the above request:
```json
{
  "domain": "dynamic-m.com",
  "subdomains": [
    "acima-minuteman-wired-wpwdkjgtjq.dynamic-m.com",
    "home-wgtpwdkzpc.dynamic-m.com"
  ]
}
```
---

