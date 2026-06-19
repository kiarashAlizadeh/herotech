--
-- PostgreSQL database dump
--

\restrict b3eP1EBGRRBrZBNu94n8B2al78Whf3s0j6xGtjjeL7rFiQ2F1qXGql8vHeeKYy6

-- Dumped from database version 18.4
-- Dumped by pg_dump version 18.4

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: auction_status; Type: TYPE; Schema: public; Owner: user
--

CREATE TYPE public.auction_status AS ENUM (
    'active',
    'ended',
    'cancelled'
);


ALTER TYPE public.auction_status OWNER TO "user";

--
-- Name: item_status; Type: TYPE; Schema: public; Owner: user
--

CREATE TYPE public.item_status AS ENUM (
    'available',
    'in_auction',
    'sold'
);


ALTER TYPE public.item_status OWNER TO "user";

--
-- Name: item_type; Type: TYPE; Schema: public; Owner: user
--

CREATE TYPE public.item_type AS ENUM (
    'common',
    'rare',
    'legendary'
);


ALTER TYPE public.item_type OWNER TO "user";

--
-- Name: transaction_type; Type: TYPE; Schema: public; Owner: user
--

CREATE TYPE public.transaction_type AS ENUM (
    'deposit',
    'purchase',
    'reserve',
    'release',
    'refund'
);


ALTER TYPE public.transaction_type OWNER TO "user";

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: auctions; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.auctions (
    id uuid DEFAULT uuidv7() NOT NULL,
    item_id uuid NOT NULL,
    seller_id uuid NOT NULL,
    status public.auction_status DEFAULT 'active'::public.auction_status NOT NULL,
    start_price bigint NOT NULL,
    highest_bid bigint,
    winner_id uuid,
    ends_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT positive_highest_bid CHECK (((highest_bid IS NULL) OR (highest_bid > 0))),
    CONSTRAINT positive_start_price CHECK ((start_price > 0))
);


ALTER TABLE public.auctions OWNER TO "user";

--
-- Name: bids; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.bids (
    id uuid DEFAULT uuidv7() NOT NULL,
    auction_id uuid NOT NULL,
    bidder_id uuid NOT NULL,
    amount bigint NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT positive_bid_amount CHECK ((amount > 0))
);


ALTER TABLE public.bids OWNER TO "user";

--
-- Name: daily_purchases; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.daily_purchases (
    guild_id uuid NOT NULL,
    date date DEFAULT CURRENT_DATE NOT NULL,
    total_spent bigint DEFAULT 0 NOT NULL,
    CONSTRAINT non_negative_spent CHECK ((total_spent >= 0))
);


ALTER TABLE public.daily_purchases OWNER TO "user";

--
-- Name: guilds; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.guilds (
    id uuid DEFAULT uuidv7() NOT NULL,
    name text NOT NULL,
    gold_balance bigint DEFAULT 0 NOT NULL,
    daily_limit bigint DEFAULT 10000 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT positive_balance CHECK ((gold_balance >= 0)),
    CONSTRAINT positive_daily_limit CHECK ((daily_limit > 0))
);


ALTER TABLE public.guilds OWNER TO "user";

--
-- Name: items; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.items (
    id uuid DEFAULT uuidv7() NOT NULL,
    name text NOT NULL,
    type public.item_type NOT NULL,
    status public.item_status DEFAULT 'available'::public.item_status NOT NULL,
    owner_id uuid NOT NULL,
    base_price bigint NOT NULL,
    list_price bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT positive_base_price CHECK ((base_price > 0)),
    CONSTRAINT positive_list_price CHECK (((list_price IS NULL) OR (list_price > 0)))
);


ALTER TABLE public.items OWNER TO "user";

--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO "user";

--
-- Name: wallet_transactions; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.wallet_transactions (
    id uuid DEFAULT uuidv7() NOT NULL,
    guild_id uuid NOT NULL,
    type public.transaction_type NOT NULL,
    amount bigint NOT NULL,
    reference_id uuid,
    description text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.wallet_transactions OWNER TO "user";

--
-- Name: auctions auctions_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.auctions
    ADD CONSTRAINT auctions_pkey PRIMARY KEY (id);


--
-- Name: bids bids_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.bids
    ADD CONSTRAINT bids_pkey PRIMARY KEY (id);


--
-- Name: daily_purchases daily_purchases_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.daily_purchases
    ADD CONSTRAINT daily_purchases_pkey PRIMARY KEY (guild_id, date);


--
-- Name: guilds guilds_name_key; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.guilds
    ADD CONSTRAINT guilds_name_key UNIQUE (name);


--
-- Name: guilds guilds_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.guilds
    ADD CONSTRAINT guilds_pkey PRIMARY KEY (id);


--
-- Name: items items_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.items
    ADD CONSTRAINT items_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: wallet_transactions wallet_transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.wallet_transactions
    ADD CONSTRAINT wallet_transactions_pkey PRIMARY KEY (id);


--
-- Name: idx_auctions_ends_at; Type: INDEX; Schema: public; Owner: user
--

CREATE INDEX idx_auctions_ends_at ON public.auctions USING btree (ends_at) WHERE (status = 'active'::public.auction_status);


--
-- Name: idx_auctions_status; Type: INDEX; Schema: public; Owner: user
--

CREATE INDEX idx_auctions_status ON public.auctions USING btree (status);


--
-- Name: idx_bids_auction; Type: INDEX; Schema: public; Owner: user
--

CREATE INDEX idx_bids_auction ON public.bids USING btree (auction_id);


--
-- Name: idx_bids_bidder; Type: INDEX; Schema: public; Owner: user
--

CREATE INDEX idx_bids_bidder ON public.bids USING btree (bidder_id);


--
-- Name: idx_items_owner; Type: INDEX; Schema: public; Owner: user
--

CREATE INDEX idx_items_owner ON public.items USING btree (owner_id);


--
-- Name: idx_items_status; Type: INDEX; Schema: public; Owner: user
--

CREATE INDEX idx_items_status ON public.items USING btree (status);


--
-- Name: idx_legendary_unique_name; Type: INDEX; Schema: public; Owner: user
--

CREATE UNIQUE INDEX idx_legendary_unique_name ON public.items USING btree (name) WHERE (type = 'legendary'::public.item_type);


--
-- Name: idx_one_active_auction_per_item; Type: INDEX; Schema: public; Owner: user
--

CREATE UNIQUE INDEX idx_one_active_auction_per_item ON public.auctions USING btree (item_id) WHERE (status = 'active'::public.auction_status);


--
-- Name: idx_one_active_bid_per_bidder; Type: INDEX; Schema: public; Owner: user
--

CREATE UNIQUE INDEX idx_one_active_bid_per_bidder ON public.bids USING btree (auction_id, bidder_id) WHERE (is_active = true);


--
-- Name: idx_wallet_tx_created; Type: INDEX; Schema: public; Owner: user
--

CREATE INDEX idx_wallet_tx_created ON public.wallet_transactions USING btree (created_at);


--
-- Name: idx_wallet_tx_guild; Type: INDEX; Schema: public; Owner: user
--

CREATE INDEX idx_wallet_tx_guild ON public.wallet_transactions USING btree (guild_id);


--
-- Name: auctions auctions_item_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.auctions
    ADD CONSTRAINT auctions_item_id_fkey FOREIGN KEY (item_id) REFERENCES public.items(id);


--
-- Name: auctions auctions_seller_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.auctions
    ADD CONSTRAINT auctions_seller_id_fkey FOREIGN KEY (seller_id) REFERENCES public.guilds(id);


--
-- Name: auctions auctions_winner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.auctions
    ADD CONSTRAINT auctions_winner_id_fkey FOREIGN KEY (winner_id) REFERENCES public.guilds(id);


--
-- Name: bids bids_auction_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.bids
    ADD CONSTRAINT bids_auction_id_fkey FOREIGN KEY (auction_id) REFERENCES public.auctions(id);


--
-- Name: bids bids_bidder_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.bids
    ADD CONSTRAINT bids_bidder_id_fkey FOREIGN KEY (bidder_id) REFERENCES public.guilds(id);


--
-- Name: daily_purchases daily_purchases_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.daily_purchases
    ADD CONSTRAINT daily_purchases_guild_id_fkey FOREIGN KEY (guild_id) REFERENCES public.guilds(id);


--
-- Name: items items_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.items
    ADD CONSTRAINT items_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.guilds(id);


--
-- Name: wallet_transactions wallet_transactions_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.wallet_transactions
    ADD CONSTRAINT wallet_transactions_guild_id_fkey FOREIGN KEY (guild_id) REFERENCES public.guilds(id);


--
-- PostgreSQL database dump complete
--

\unrestrict b3eP1EBGRRBrZBNu94n8B2al78Whf3s0j6xGtjjeL7rFiQ2F1qXGql8vHeeKYy6

