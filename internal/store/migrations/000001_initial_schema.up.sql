-- CREATE TABLES

-- Tabela: utilizadores
CREATE TABLE utilizadores (
                              id SERIAL PRIMARY KEY,
                              nome VARCHAR(100) NOT NULL UNIQUE,
                              password VARCHAR(255) NOT NULL,
                              tipo VARCHAR(50) NOT NULL CHECK (tipo IN ('Aluno', 'Professor', 'Admin')), -- Exemplo de tipos
                              datanascimento DATE,
                              criacao TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabela: tema
CREATE TABLE tema (
                      id SERIAL PRIMARY KEY,
                      nome VARCHAR(100) NOT NULL UNIQUE,
                      descricao TEXT,
                      criacao TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
                      ativo BOOLEAN NOT NULL DEFAULT TRUE
);

-- Tabela: quizzes
CREATE TABLE quizzes (
                         id SERIAL PRIMARY KEY,
                         nome VARCHAR(100) NOT NULL,
                         tema_id INTEGER NOT NULL REFERENCES tema(id) ON DELETE RESTRICT,
                         UNIQUE (nome, tema_id),
                         ativo BOOLEAN NOT NULL DEFAULT TRUE
);

-- Tabela: perguntas
CREATE TABLE perguntas (
                           id SERIAL PRIMARY KEY,
                           assunto VARCHAR(100),
                           corpo TEXT NOT NULL,
                           explicacao TEXT,
                           quizz_id INTEGER NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE
);

-- Tabela: respostas
CREATE TABLE respostas (
                           id SERIAL PRIMARY KEY,
                           corpo TEXT NOT NULL,
                           correta BOOLEAN NOT NULL DEFAULT FALSE,
                           pergunta_id INTEGER NOT NULL REFERENCES perguntas(id) ON DELETE CASCADE
);

-- Tabela: submissao
CREATE TABLE submissao (
                           id SERIAL PRIMARY KEY,
                           datahora TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                           pontuacao NUMERIC(5, 2) NOT NULL CHECK (pontuacao >= 0 AND pontuacao <= 100),
                           utilizador_id INTEGER NOT NULL REFERENCES utilizadores(id) ON DELETE RESTRICT,
                           quizz_id INTEGER NOT NULL REFERENCES quizzes(id) ON DELETE RESTRICT,
                           UNIQUE (utilizador_id, quizz_id, datahora) -- Um utilizador pode fazer o mesmo quizz várias vezes
);

-- Tabela: dificuldades
CREATE TABLE dificuldades (
                              id SERIAL PRIMARY KEY,
                              assunto VARCHAR(100),
                              submissao_id INTEGER NOT NULL REFERENCES submissao(id) ON DELETE CASCADE
);

-- Tabela: respostas_dadas
CREATE TABLE respostas_dadas (
                                 id SERIAL PRIMARY KEY,
                                 submissao_id INTEGER NOT NULL REFERENCES submissao(id) ON DELETE CASCADE,
                                 pergunta_id INTEGER NOT NULL REFERENCES perguntas(id) ON DELETE RESTRICT, -- Mantém a referência à pergunta
                                 resposta_id INTEGER REFERENCES respostas(id) ON DELETE RESTRICT, -- Resposta escolhida
                                 correta_na_submissao BOOLEAN, -- Se a resposta dada estava correta (já calculada)
                                 UNIQUE (submissao_id, pergunta_id)
);