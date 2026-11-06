-- AI assistant (Aleph) conversation tracking.
-- company_id / user_id use VARCHAR so they accept the same string ids the
-- other tables receive from the API.

-- One row per conversation (a chat thread of one user inside one company).
CREATE TABLE IF NOT EXISTS public.tbl_ai_conversation (
    conversation_id UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      VARCHAR(100) NOT NULL,
    user_id         VARCHAR(128) NOT NULL,
    platform        VARCHAR(40)  NOT NULL,
    area            VARCHAR(60)  NOT NULL DEFAULT 'general',
    title           VARCHAR(200),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ai_conversation_company_user
    ON public.tbl_ai_conversation (company_id, user_id, updated_at DESC);

-- One row per message (the user's question or the assistant's answer).
CREATE TABLE IF NOT EXISTS public.tbl_ai_message (
    message_id      UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID         NOT NULL
                    REFERENCES public.tbl_ai_conversation (conversation_id)
                    ON DELETE CASCADE,
    company_id      VARCHAR(100) NOT NULL,
    user_id         VARCHAR(128) NOT NULL,
    role            VARCHAR(20)  NOT NULL CHECK (role IN ('user', 'assistant')),
    content         TEXT         NOT NULL,
    on_topic        BOOLEAN,
    model           VARCHAR(60),
    input_tokens    INTEGER      NOT NULL DEFAULT 0,
    output_tokens   INTEGER      NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX IF NOT EXISTS idx_ai_message_conversation
    ON public.tbl_ai_message (conversation_id, created_at);

-- Used by the daily limits (messages per user per day).
CREATE INDEX IF NOT EXISTS idx_ai_message_company_user_day
    ON public.tbl_ai_message (company_id, user_id, created_at);
