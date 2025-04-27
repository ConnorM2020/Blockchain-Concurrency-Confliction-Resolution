# Blockchain Concurrency Confliction Resolution

An MEng Final Year Project - Blockchain platform for resolving concurrency conflicts using sharding and MVCC principles, with full visualisation, benchmarking, and persistent Firebase logging.

---

## Project Overview

This project showcases a scalable, conflict-resilient blockchain system built with a Go-based backend and a React-based frontend. It supports:

- Concurrent transaction execution
- Sharding with MVCC conflict resolution
- Interactive dashboard with ReactFlow
- Firebase integration for persistent logging and recovery
- Performance benchmarking: execution time, finality, propagation latency, TPS

---

## Why Use Sharding?

> **Scalability**: Transactions execute in parallel across shards.  
> **Performance**: Lower average execution time for sharded workloads.  
> **Conflict Isolation**: Sharding reduces concurrency bottlenecks.

---

##  Project Structure

```
Blockchain-Concurrency-Confliction-Resolution/
├── Blockchain_Codebase/         # Go-based blockchain logic
│   ├── firebase.go              # Firebase integration
│   ├── block.go                 # Core blockchain types & logic
│   ├── working_main.go         # Main server logic
│   ├── sharding.go             # Shard creation and mapping
│   └── blockchain-visualizer/  # ReactFlow UI frontend
├── chaincode/                  # Fabric chaincode (smart contracts)
├── fabric-samples/             # Hyperledger Fabric network setup
├── fablo-target/               # Artifacts for Docker-based Fabric startup
├── backup/                     # Historical copies and recovery
├── README.md                   # This file
```

---

##  Transaction Metrics (Visualised)

The system logs each transaction with detailed metrics:

| Metric               | Description                                             |
|---------------------|---------------------------------------------------------|
| `txID`              | Unique transaction identifier                          |
| `source → target`   | Block-to-block routing                                 |
| `type`              | `Sharded` or `Non-Sharded`                             |
| `execTime` (ms)     | Time to execute the transaction                        |
| `finality` (ms)     | Time from submission to full confirmation              |
| `propagation` (ms)  | Network latency between blocks                         |
| `TPS`               | Transactions per second at the time of submission      |
| `timestamp`         | Time of submission                                     |

---

## 📊 Visual Dashboard

Transactions are pulled directly from **Firebase Firestore** and shown in ascending timestamp order:

## Firebase Integration

All transaction logs are saved to Firestore (`transactions` collection), including performance details and timestamps for analytics. This enables persistence across system restarts and supports dashboard loading from historical data.

Sample Firestore entry:
```json
{
  "txID": "tx-17443889221333985170",
  "source": 1,
  "target": 2,
  "type": "Sharded",
  "execTime": 1000.744,
  "finality": 1000.744,
  "tps": 1.0,
  "timestamp": "2025-04-05T22:40:22",
  "message": "Transaction Data",
  "propagation": 35
}
```
---

## Setup Instructions

### Prerequisites
- Go >= 1.23.2
- Node.js + npm
- Docker + Docker Compose (for Fabric)
- Firebase Admin SDK JSON (ignored in Git)
- WSL2 (if on Windows)

### Cloning the Repo
```bash
git clone https://gitlab.eeecs.qub.ac.uk/40295919/csc4006-project
cd Blockchain-Concurrency-Confliction-Resolution
```

### Backend (Go)
```bash
cd Blockchain_Codebase
./startFabric.sh
./blockchain_app --server -process
```

### Frontend (ReactFlow)
```bash
cd Blockchain_Codebase/blockchain-visualizer
npm install
npm run dev
```

---

## Contributors
- 40295919  (📌 Main Developer, UI + Backend)

---

## 📄 License
This project is licensed under the **MIT License**. Feel free to adapt and reuse.

## 📬 Contact
- GitHub: [ConnorM2020](https://github.com/ConnorM2020)
- Email: Available via GitHub Profile

---


=======
# csc4006-project
=======


## Getting started

To make it easy for you to get started with GitLab, here's a list of recommended next steps.

Already a pro? Just edit this README.md and make it your own. Want to make it easy? [Use the template at the bottom](#editing-this-readme)!

## Add your files

- [ ] [Create](https://docs.gitlab.com/ee/user/project/repository/web_editor.html#create-a-file) or [upload](https://docs.gitlab.com/ee/user/project/repository/web_editor.html#upload-a-file) files
- [ ] [Add files using the command line](https://docs.gitlab.com/ee/gitlab-basics/add-file.html#add-a-file-using-the-command-line) or push an existing Git repository with the following command:

```
cd existing_repo
<<<<<<< HEAD
git remote add origin https://gitlab2.eeecs.qub.ac.uk/40295919/csc4006-project.git
=======
git remote add origin https://gitlab.eeecs.qub.ac.uk/40295919/csc4006-project.git
>>>>>>> b229c76263cec0f2950aec8dd05bd27709ea302c
git branch -M main
git push -uf origin main
```

## Integrate with your tools

<<<<<<< HEAD
- [ ] [Set up project integrations](https://gitlab2.eeecs.qub.ac.uk/40295919/csc4006-project/-/settings/integrations)
=======
- [ ] [Set up project integrations](https://gitlab.eeecs.qub.ac.uk/40295919/csc4006-project/-/settings/integrations)
>>>>>>> b229c76263cec0f2950aec8dd05bd27709ea302c

## Collaborate with your team

- [ ] [Invite team members and collaborators](https://docs.gitlab.com/ee/user/project/members/)
- [ ] [Create a new merge request](https://docs.gitlab.com/ee/user/project/merge_requests/creating_merge_requests.html)
- [ ] [Automatically close issues from merge requests](https://docs.gitlab.com/ee/user/project/issues/managing_issues.html#closing-issues-automatically)
- [ ] [Enable merge request approvals](https://docs.gitlab.com/ee/user/project/merge_requests/approvals/)
- [ ] [Set auto-merge](https://docs.gitlab.com/ee/user/project/merge_requests/merge_when_pipeline_succeeds.html)

## Test and Deploy

Use the built-in continuous integration in GitLab.

- [ ] [Get started with GitLab CI/CD](https://docs.gitlab.com/ee/ci/quick_start/index.html)
- [ ] [Analyze your code for known vulnerabilities with Static Application Security Testing (SAST)](https://docs.gitlab.com/ee/user/application_security/sast/)
- [ ] [Deploy to Kubernetes, Amazon EC2, or Amazon ECS using Auto Deploy](https://docs.gitlab.com/ee/topics/autodevops/requirements.html)
- [ ] [Use pull-based deployments for improved Kubernetes management](https://docs.gitlab.com/ee/user/clusters/agent/)
- [ ] [Set up protected environments](https://docs.gitlab.com/ee/ci/environments/protected_environments.html)

***

# Editing this README

When you're ready to make this README your own, just edit this file and use the handy template below (or feel free to structure it however you want - this is just a starting point!). Thanks to [makeareadme.com](https://www.makeareadme.com/) for this template.

## Suggestions for a good README

Every project is different, so consider which of these sections apply to yours. The sections used in the template are suggestions for most open source projects. Also keep in mind that while a README can be too long and detailed, too long is better than too short. If you think your README is too long, consider utilizing another form of documentation rather than cutting out information.

## Name
Choose a self-explaining name for your project.

## Description
Let people know what your project can do specifically. Provide context and add a link to any reference visitors might be unfamiliar with. A list of Features or a Background subsection can also be added here. If there are alternatives to your project, this is a good place to list differentiating factors.

## Badges
On some READMEs, you may see small images that convey metadata, such as whether or not all the tests are passing for the project. You can use Shields to add some to your README. Many services also have instructions for adding a badge.

## Visuals
Depending on what you are making, it can be a good idea to include screenshots or even a video (you'll frequently see GIFs rather than actual videos). Tools like ttygif can help, but check out Asciinema for a more sophisticated method.

## Installation
Within a particular ecosystem, there may be a common way of installing things, such as using Yarn, NuGet, or Homebrew. However, consider the possibility that whoever is reading your README is a novice and would like more guidance. Listing specific steps helps remove ambiguity and gets people to using your project as quickly as possible. If it only runs in a specific context like a particular programming language version or operating system or has dependencies that have to be installed manually, also add a Requirements subsection.

## Usage
Use examples liberally, and show the expected output if you can. It's helpful to have inline the smallest example of usage that you can demonstrate, while providing links to more sophisticated examples if they are too long to reasonably include in the README.

## Support
Tell people where they can go to for help. It can be any combination of an issue tracker, a chat room, an email address, etc.

## Roadmap
If you have ideas for releases in the future, it is a good idea to list them in the README.

## Contributing
State if you are open to contributions and what your requirements are for accepting them.

For people who want to make changes to your project, it's helpful to have some documentation on how to get started. Perhaps there is a script that they should run or some environment variables that they need to set. Make these steps explicit. These instructions could also be useful to your future self.

You can also document commands to lint the code or run tests. These steps help to ensure high code quality and reduce the likelihood that the changes inadvertently break something. Having instructions for running tests is especially helpful if it requires external setup, such as starting a Selenium server for testing in a browser.

## Authors and acknowledgment
Show your appreciation to those who have contributed to the project.

## License
For open source projects, say how it is licensed.

## Project status
If you have run out of energy or time for your project, put a note at the top of the README saying that development has slowed down or stopped completely. Someone may choose to fork your project or volunteer to step in as a maintainer or owner, allowing your project to keep going. You can also make an explicit request for maintainers.
<<<<<<< HEAD
>>>>>>> 0ba6a8c3167c503c5ba252fe804b97b65da2c599
=======
>>>>>>> b229c76263cec0f2950aec8dd05bd27709ea302c
