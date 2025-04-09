# GWI - Jedi Team - Engineering Challenge

Welcome to the engineering challenge for the Jedi Team at GWI!

This task is designed to help us understand how you approach software engineering problems and apply your skills in a real-world-inspired scenario. It focuses on backend engineering using **Go**, with optional extensions into **AI/LLMs**, **product thinking**, and **system design**. The Jedi team mainly works on and evolves the AI infrastructure of the company, so this exercise has a strong focus on that.

While the base functionality is straightforward, we encourage you to go beyond the minimum requirements — creativity, thoughtful design, and clean code are all appreciated.

## 🧪 Core Requirements

You are going to create a **chatbot** that helps GWI's clients answer questions based on market research data. Another tool has converted GWI's data into a **natural language** format and stored it in a database. You can find the data in `data.md`. You should use this data to answer users' questions.

Build a web server in **Go** that exposes this chat functionality (you decide the communication method and the necessary endpoints). The discussion within the chat should be persisted, and the user should be able to continue the conversation from where it was left off. A single user can open multiple chats.

## 🌟 Optional Enhancements

- If the answer to the user's question is not found in the data, the chatbot should decline to answer.
- The user can give negative feedback on a message.
- The chat should have an auto-generated title.
- Include a **Dockerfile** and a **Makefile** or **Taskfile** to simplify local development.
- Explain in the README how to run the application and the assumptions you made.

## 🧩 Submission

Just fork the current repository and send it to us!

Good luck, potential colleague!
