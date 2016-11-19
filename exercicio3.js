// LE01ESTADO
db.runCommand({
    collMod: "LE01ESTADO",
    validator: {
        $and: [
            {_id: { Sigla: { $exists: true }}},
            {Nome: { $exists: true } }
        ]
    }
})

// LE02CIDADE
db.runCommand({
    collMod: "LE02CIDADE",
    validator: {
        $and: [
            {_id:{Nome:{$exists: true}, SiglaEstado:{$exists: true}},
            Populacao: {$exists: true}}
        ]
    }
})

// LE03ZONA
db.runCommand({
    collMod: "LE03ZONA",
    validator: {
        $and: [
            {_id: {NroZona: {$exists: true}},
            NroDeUrnasReservas: {$exists: true}}
        ]
    }
})

// LE04BAIRRO
db.runCommand({
    collMod: "LE04BAIRRO",
    validator: {
        $and: [
            {_id: {Nome: {$exists: true}, NomeCidade: {$exists: true}, SiglaEstado: {$exists: true}},
            NroZona: {$exists:true}}
        ]
    }
})

// LE05URNA
db.runCommand({
    collMod: "LE05URNA",
    validator: {
        $and: [
            {_id: {NSerial: {$exists: true}},
            Estado: {$in: ["funcional", "manutencao"]}}
        ]
    }
})

// LE06SESSAO
db.runCommand({
    collMod: "LE06SESSAO",
    validator: {
        $and: [
            {_id: {NroSessao: {$exists: true}},
            NSerial: {$exists: true}}
        ]
    }
})

// LE07PARTIDO
db.runCommand({
    collMod: "LE07PARTIDO",
    validator: {
        $and: [
            {_id: {Sigla: {$exists: true}},
            Nome: {$exists: true}}
        ]
    }
})

// LE08CANDIDATO
db.runCommand({
    collMod: "LE08CANDIDATO",
    validator: {
        $and: [
            {_id: {NroCand: {$exists: true}},
            Tipo: {$in: ["politico", "especial"]},
            Nome: {$exists: true}}
        ]
    }
})

// LE09CARGO
db.runCommand({
    collMod: "LE09CARGO",
    validator: {
        $and: [
            {_id: {CodCargo: {$exists: true}},
            PossuiVice: {$exists: true, $in:[0, 1]},
            NroDeCadeiras: {$exists: true, $gt: 0},
            Esfera: {$exists: true, $in: ["F", "E", "M"]},
            NomeDescritivo: {$exists: true},
            AnosMandato: {$exists: true, $gt: 0},
            AnoBase: {$exists: true, $gt: 1985, $lt: 2100}},
            {$or: [
                {Esfera: {$eq: "F"}, NomeCidade: {$exists: false}, SiglaEstado: {$exists: false}},
                {Esfera: {$eq: "E"}, NomeCidade: {$exists: false}, SiglaEstado: {$exists: true}},
                {Esfera: {$eq: "M"}, NomeCidade: {$exists: true}, SiglaEstado: {$exists: false}}
            ]}
        ]
    }
})

// LE10CANDIDATURA
db.runCommand({
    collMod: "LE10CANDIDATURA",
    validator: {
        $and: [
            {_id: {Reg: {$exists: true}},
            CodCargo: {$exists: true},
            Ano: {$exists: true, $gt: 1985, $lt: 2100},
            NroCand: {$exists: true},
            Nome: {$exists: true}}
        ]
    }
})

// LE11PLEITO
db.runCommand({
    collMod: "LE11PLEITO",
    validator: {
        $and: [
            {_id: {NroSessao: {$exists: true}, NroZona: {$exists: true}, CodCargo: {$exists: true}, Ano: {$exists: true}, NroCand: {$exists: true}},
            CodCargo: {$exists: true},
            Ano: {$exists: true, $gt: 1985, $lt: 2100},
            NroCand: {$exists: true},
            Nome: {$exists: true}}
        ]
    }
})

// LE12PESQUISA
db.runCommand({
    collMod: "LE12PESQUISA",
    validator: {
        $and: [
            {_id: {RegPesquisa: {$exists: true}},
            PeriodoInicio: {$exists: true},
            PeriodoFim: {$exists: true, $gt: 1985, $lt: 2100}}
        ]
    }
})

// LE13INTENCAODEVOTO
db.runCommand({
    collMod: "LE13INTENCAODEVOTO",
    validator: {
        $and: [
            {_id: {RegPesquisa: {$exists: true}, RegCandid: {$exists: true}},
            Total: {$exists: true}}
        ]
    }
})