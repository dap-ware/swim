
<h1 align="center">
-- Swim --
<br>Certificate Transparency Log
<br>Analysis Framework
</h1>

<p align="center">
  <img src="resources/logo.png" alt="Swim Logo" width="400">
</p>

---

# Table of Contents
- [What is Swim?](#what-is-swim)
- [Under Active Development](#under-active-development)
- [Installation](#installation)
  - [Prerequisites](#prerequisites)
  - [Installation Methods](#installation-methods)
- [Generating a Self-Signed Certificate](#generating-a-self-signed-certificate)
- [Why You Need a Certificate](#why-you-need-a-certificate)
- [Usage](#usage)
- [API Reference](#api-reference)
  - [Fetch Domain Names](#fetch-domain-names-apex-domains)
  - [Fetch Domain Specific Cert Update Event Data](#fetch-domain-specific-cert-update-event-data)
  - [Fetch Subdomains](#fetch-subdomains)

---

# What is Swim?

> Swim is an application written in Go, specifically developed to interface with the Calidog CertStream service through websockets. It is designed for real-time processing and analysis of SSL/TLS certificate transparency logs. The application employs a stream processing approach to continuously ingest and parse certificate transparency logs, extracting domain-specific information with high efficiency.
> 
> The core of Swim is its data processing pipeline, which includes:
> - Real-time ingestion of log data via websockets.
> - Parsing and normalization of log data to extract relevant information such as domain names, certificate issuance details, and more.
> - Efficient storage of processed data in a SQLite database, ensuring optimized data retrieval and query performance.
>
> A key technical component of Swim is the implementation of a RESTful API built using the Gin web framework. This API provides programmatic access to the processed data, allowing users to perform complex queries with ease. The API supports various operations such as:
> - Retrieval of detailed domain information.
> - Access to historical certificate event data.
> - Exploration of subdomain structures and related metadata.
>
> These functionalities are particularly valuable for security analysts and researchers who need timely access to detailed information about SSL/TLS certificate issuance and domain changes. 
>
> The choice of Gin for the API framework is due to its high performance, low memory footprint, and scalability. This ensures that Swim can handle a large volume of requests while maintaining responsiveness and efficiency.

---

## Under Active Development

> Swim is in a phase of active development, with an ongoing focus on enhancing its feature set. Future updates planned for Swim include:
> - Development of a web-based user interface (UI) to provide a more interactive and intuitive experience for data visualization and analysis.
> - The UI aims to simplify the exploration and interpretation of complex datasets, making it more accessible for users engaged in security research and monitoring.
>
> These enhancements are aligned with the goal of evolving Swim into a more comprehensive tool for cybersecurity and IT professionals.

---

# **Installation**


> ### **Prerequisites**
>
> 1. Go installed on system
> 2. That's it...
>
> ### **Using Go Install**
>
> - If you have Go installed and configured (with `GOPATH` set up), you can directly install Swim using the `go install` command:
>
>     ```bash
>    go install github.com/dap-ware/swim@latest
>     ```
> 
> ### **Using git clone and building from source**
>
>
> - Alternatively, you can clone the repository and build Swim manually. This is a good option if you want to work with the source code or contribute to the project:
>
> 1. Clone the repository:
> 
>    ```bash
>    git clone https://github.com/dap-ware/swim.git
>    cd swim
>    ```
> 2. Build the application using Go:
> 
>    ```bash
>    go build .
>    ```
>    
>    - This will compile the Swim application and create an executable file in the current directory.

---

# Generating a Self-Signed Certificate

> To generate a self-signed certificate and key with OpenSSL, follow these steps:
> 
> 1. First, make sure you have OpenSSL installed on your machine. If you don't have it installed, you can download it from the [official OpenSSL website](https://www.openssl.org/source/).
>
> 2. Open a terminal and navigate to your project directory.
>
> 3. Create a `cert/` directory in your project directory if it doesn't already exist:
>
>     ```bash
>     mkdir -p cert/
>     ```
>
> 4. Navigate to the `cert/` directory:
>
>     ```bash
>     cd cert/
>     ```
>
> 5. Generate a new self-signed certificate and key with the following command:
>
>     ```bash
>     openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
>     ```
>
>     This command generates a new RSA key (`key.pem`) and a self-signed certificate (`cert.pem`) that are valid for 365 days. The `-nodes` option means that the key will not be encrypted.
> 
> 6. You should now have a `cert.pem` and `key.pem` file in your `cert/` directory. You can verify this with the following command:
>
>     ```bash
>     ls -l
>     ```
>
>     You should see `cert.pem` and `key.pem` in the output.
>
> Remember, self-signed certificates should only be used for testing and development purposes. For production environments, you should use certificates issued by a trusted Certificate Authority.

---

# Why You Need a SSL/TLS Certificate

> When you're running a server over HTTPS, you need a SSL/TLS certificate. This certificate serves two main purposes:
>
>> - **Identity Verification**: The certificate verifies the identity of the server. When a client (like a web browser) connects to an HTTPS server, the server sends its certificate to the client. The client can then verify the identity of the server using the information in the certificate.
>>
>> - **Data Encryption**: The certificate is used to establish a secure encrypted connection between the client and the server. This ensures that the data transmitted between the client and the server is private and secure.
> 
> In a production environment, you would typically use a certificate issued by a trusted Certificate Authority (CA). The CA verifies your identity and issues a certificate that browsers and other clients trust. For testing and development, you can use a self-signed certificate. 
>
> A self-signed certificate is a certificate that is not signed by a trusted CA. This means that browsers and other clients won't trust it by default and will show a warning when connecting to your server. However, for testing and development, this is usually acceptable.
>
> Please note that the self-signed certificate generated by the instructions above is only valid for 365 days. After that, you'll need to generate a new one.

---

# **Usage**

> ### **Running Swim**
>
> **To run Swim with custom settings, modify config/config.yaml, then run:**
> ```bash
> ./swim
> ```
>
> **To display help information:**
> ```bash
> ./swim -h
> ```
>
> **Making a request to the api to ensure everything is working (while swim is running)**
> ```bash
> curl https://localhost:8080/v1/domains?page=1&size=1000
> ```
>

---

# **API Reference**

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
**Endpoint**: `GET /v1/get/cert-updates?page=1&size=100`
- This endpoint retrieves certificate update event data for domains. The data includes details such as domain names, their apex status, parent domains, SSL certificate information, and more.

#### **Query Parameters**
- `page`: Page number for pagination (default: 1)
- `size`: Number of cert-update records per page (default: 1000)

#### **Example Request**
- To fetch the first page of domain event data with 2 records per page:
```bash
GET http://localhost:8080/v1/cert-updates?page=1&size=2
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
**Endpoint**: `GET /v1/get/subdomains/:domain`
- This endpoint retrieves a list of subdomains for a given domain name. It is useful for identifying all subdomains associated with a specific apex domain, which can be crucial for domain management and security analysis.

#### **Path Parameters**
- `domain`: The domain name for which subdomains are to be fetched. For example, `dynamic-m.com`.

#### **Example Request**
To fetch subdomains for the domain `dynamic-m.com`:
```bash
GET http://localhost:8080/v1/subdomains/dynamic-m.com
```

#### **Example Response**
- The response is streamed as JSON, here is the example response from the above request:
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

