<div align="center"><p>
<a href="#what-is-ynet-studio">Ynet Studio</a> •
<a href="#feature-list">Feature list</a> •
<a href="#quickstart">Quickstart</a> •
<a href="#developer-guide">Developer Guide</a>
</p>
<p>
  <img alt="License" src="https://img.shields.io/badge/license-apache2.0-blue.svg">
  <img alt="Go Version" src="https://img.shields.io/badge/go-%3E%3D%201.23.4-blue">
</p>

English | [中文](README.zh_CN.md)

</div>

## What is Ynet Studio?

Ynet Studio is an all-in-one AI agent development tool. Providing the latest large models and tools, various development modes and frameworks, Ynet Studio offers the most convenient AI agent development environment, from development to deployment. 

* **Provides all core technologies needed for AI agent development**: prompt, RAG, plugin, workflow, enabling developers to focus on creating the core value of AI.
* **Ready to use for professional AI agent development at the lowest cost**: Ynet Studio provides developers with complete app templates and build frameworks, allowing you to quickly construct various AI agents and turn creative ideas into reality.

Ynet Studio, derived from the "Ynet Development Platform" which has served tens of thousands of enterprises and millions of developers, we have made its core engine completely open. It is a one-stop visual development tool for AI Agents that makes creating, debugging, and deploying AI Agents unprecedentedly simple. Through Ynet Studio's visual design and build tools, developers can quickly create and debug agents, apps, and workflows using no-code or low-code approaches, enabling powerful AI app development and more customized business logic. It's an ideal choice for building low-code AI products tailored . Ynet Studio aims to lower the threshold for AI agent development and application, encouraging community co-construction and sharing for deeper exploration and practice in the AI field.

The backend of Ynet Studio is developed using Golang, the frontend uses React + TypeScript, and the overall architecture is based on microservices and built following domain-driven design (DDD) principles. Provide developers with a high-performance, highly scalable, and easy-to-customize underlying framework to help them address complex business needs.
## Feature list
| **Module** | **Feature** |
| --- | --- |
| Model service | Manage the model list, integrate services such as OpenAI and Volcengine |
| Build agent | * Build, publish, and manage agent <br> * Support configuring workflows, knowledge bases, and other resources |
| Build apps | * Create and publish apps <br> * Build business logic through workflows |
| Build a workflow | Create, modify, publish, and delete workflows |
| Develop resources | Support creating and managing the following resources: <br> * Plugins <br> * Knowledge bases <br> * Databases <br> * Prompts |
| API and SDK | * Create conversations, initiate chats, and other OpenAPI <br> * Integrate agents or apps into your own app through Chat SDK |

## Quickstart
Learn how to obtain and deploy the open-source version of Ynet Studio, quickly build projects, and experience Ynet Studio's open-source version.

Environment requirements:

* Before installing Ynet Studio, please ensure that your machine meets the following minimum system requirements: 2 Core、4 GB
* Pre-install Docker and Docker Compose, and start the Docker service.

Deployment steps:

1. Retrieve the source code.
   ```Bash
   # Clone code
   git clone http://git.ynet.io/Cheetah/Develop/ai/agent/ynet/ynet-studio.git
   ```

2. Configure the model.
   1. Copy the template files of the doubao-seed-1.6 model from the template directory and paste them into the configuration file directory.
      ```Bash
      cd ynet-studio
      # Copy model configuration template
      cp backend/conf/model/template/model_template_ark_doubao-seed-1.6.yaml backend/conf/model/ark_doubao-seed-1.6.yaml
      ```

   2. Modify the template file in the configuration file directory.
      1. Enter the directory `backend/conf/model`. Open the file `ark_doubao-seed-1.6.yaml`.
      2. Set the fields `id`, `meta.conn_config.api_key`, `meta.conn_config.model`, and save the file.
         * **id**: The model ID in Ynet Studio, defined by the developer, must be a non-zero integer and globally unique. Agents or workflows call models based on model IDs. For models that have already been launched, do not modify their IDs; otherwise, it may result in model call failures.
         * **meta.conn_config.api_key**: The API Key for the model service. In this example, it is the API Key for Ark API Key. For more information, see [Get Volcengine Ark API Key](https://www.volcengine.com/docs/82379/1541594) or [Get BytePlus ModelArk API Key](https://docs.byteplus.com/en/docs/ModelArk/1361424).
         * **meta.conn_config.model**: The Model name for the model service. In this example, it refers to the Model ID or Endpoint ID of Ark. For more information, see [Get Volcengine Ark Model ID](https://www.volcengine.com/docs/82379/1513689) / [Get Volcengine Ark Endpoint ID](https://www.volcengine.com/docs/82379/1099522) or  [Get BytePlus ModelArk Model ID](https://docs.byteplus.com/en/docs/ModelArk/model_id) / [Get BytePlus ModelArk Endpoint ID](https://docs.byteplus.com/en/docs/ModelArk/1099522).  
         > For users in China, you may use Volcengine Ark; for users outside China, you may use BytePlus ModelArk instead.
3. Deploy and start the service.
   When deploying and starting Ynet Studio for the first time, it may take a while to retrieve images and build local images. Please be patient. During deployment, you will see the following log information. If you see the message "Container ynet-server Started," it means the Ynet Studio service has started successfully.
   ```Bash
   # Start the service
   cd docker
   cp .env.example .env
   docker compose up -d
   ```
4. After starting the service, you can open Ynet Studio by accessing `http://localhost:8888/` through your browser.


## Developer Guide

* **Project Configuration**:
   * Model Configuration: Before deploying the open-source version of Ynet Studio, you must configure the model service. Otherwise, you cannot select models when building agents, workflows, and apps.
   * Plugin Configuration: To use official plugins from the plugin store, you must first configure the plugins and add the authentication keys for third-party services.
   * Basic Component Configuration: Learn how to configure components such as image uploaders to use functions like image uploading in Ynet Studio .
* API Reference: The Ynet Studio Community Edition API and Chat SDK are authenticated using Personal Access Token, providing APIs for conversations and workflows.
* Development Guidelines:
   * Project Architecture: Learn about the technical architecture and core components of the open-source version of Ynet Studio.
   * Code Development and Testing: Learn how to perform secondary development and testing based on the open-source version of Ynet Studio.
   * Troubleshooting: Learn how to view container states and system logs.

## Using the open-source version of Ynet Studio
> Please note that certain features, such as tone customization, are limited to the commercial version. Differences between the open-source and commercial versions can be found in the **Feature List**.

## License
This project uses the Apache 2.0 license. For details, please refer to the [LICENSE](LICENSE-APACHE) file.
## Community contributions
We welcome community contributions. For contribution guidelines, please refer to [CONTRIBUTING](CONTRIBUTING.md) and [Code of conduct](CODE_OF_CONDUCT.md). We look forward to your contributions!
## Security and privacy
If you discover potential security issues in the project, or believe you may have found a security issue, please notify the security team in time.

## Acknowledgments
Thank you to all the developers and community members who have contributed to the Ynet Studio project. Special thanks:

* The [Eino](https://github.com/cloudwego/eino) framework team - providing powerful support for Ynet Studio's agent and workflow runtime engines, model abstractions and implementations, and knowledge base indexing and retrieval
* The [FlowGram](https://github.com/bytedance/flowgram.ai) team - providing a high-quality workflow building engine for Ynet Studio's frontend workflow canvas editor
* The [Hertz](https://github.com/cloudwego/hertz) team - Go HTTP framework with high-performance and strong-extensibility for building micro-services
* All users who participated in testing and feedback
