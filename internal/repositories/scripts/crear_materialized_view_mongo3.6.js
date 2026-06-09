var currentDate = new Date().toISOString().slice(0, 10);

var pipeline = [
  // 1. Funcionario Permanente Lookup
  {
    $lookup: {
      from: "permanents",
      let: { permId: "$permanent" },
      pipeline: [
        {
          $match: {
            $expr: {
              $and: [
                { $eq: ["$_id", "$$permId"] },
                { $eq: ["$active", true] },
                { $eq: [{ $type: "$disconnection" }, "missing"] },
              ],
            },
          },
        },
        {
          $lookup: {
            from: "items",
            localField: "item",
            foreignField: "_id",
            as: "item_data",
          },
        },
        { $unwind: { path: "$item_data", preserveNullAndEmptyArrays: true } },
        {
          $project: {
            active: 1,
            disconnection: 1,
            "item_data.unit": 1,
            "item_data.denomination": 1,
            "item_data.active": 1,
            "item_data.dependence": 1,
            "item_data.position": 1,
          },
        },
      ],
      as: "funcionario_permanente",
    },
  },
  // 2. Funcionario Eventual Lookup
  {
    $lookup: {
      from: "eventuals",
      let: { eventId: "$eventual" },
      pipeline: [
        {
          $match: {
            $expr: {
              $and: [
                { $eq: ["$_id", "$$eventId"] },
                { $in: ["$state", ["APROBADO", "MODIFICADO"]] },
                { $gte: ["$endDate", currentDate] },
              ],
            },
          },
        },
        {
          $lookup: {
            from: "eventual_units",
            localField: "unit",
            foreignField: "_id",
            as: "unit_data",
          },
        },
        { $unwind: { path: "$unit_data", preserveNullAndEmptyArrays: true } },
        {
          $lookup: {
            from: "eventual_scales",
            localField: "scale",
            foreignField: "_id",
            as: "scale_data",
          },
        },
        { $unwind: { path: "$scale_data", preserveNullAndEmptyArrays: true } },
        {
          $project: {
            endDate: 1,
            "unit_data.unit": 1,
            "unit_data.name": 1,
            "unit_data.description": 1,
            "unit_data.management": 1,
            "scale_data.position": 1,
          },
        },
      ],
      as: "funcionario_eventual",
    },
  },
  // 3. Filtrar los que tienen algún contrato
  {
    $match: {
      $or: [{ funcionario_permanente: { $ne: [] } }, { funcionario_eventual: { $ne: [] } }],
    },
  },
  // 4. Aplanar campos de contrato
  {
    $addFields: {
      funcionario_permanente: { $arrayElemAt: ["$funcionario_permanente", 0] },
      funcionario_eventual: { $arrayElemAt: ["$funcionario_eventual", 0] },
    },
  },
  // 5. Lookup Senadores
  {
    $lookup: {
      from: "senadores",
      localField: "ci",
      foreignField: "ci",
      as: "datos_senador",
    },
  },
  {
    $addFields: {
      datos_senador: { $arrayElemAt: ["$datos_senador", 0] },
    },
  },
  // 6. Lookup Titular Relacionado
  {
    $lookup: {
      from: "senadores",
      let: { my_ci: "$ci" },
      pipeline: [
        {
          $match: {
            $expr: {
              $and: [{ $eq: ["$suplente", "$$my_ci"] }, { $ne: ["$suplente", null] }, { $ne: ["$suplente", ""] }],
            },
          },
        },
        { $project: { ci: 1 } },
      ],
      as: "titular_relacionado",
    },
  },
  {
    $addFields: {
      titular_relacionado: { $arrayElemAt: ["$titular_relacionado", 0] },
    },
  },
  // 7. Proyección Final
  {
    $project: {
      _id: 1,
      firstname: 1,
      secondname: 1,
      lastname: 1,
      surname: 1,
      ci: 1,
      cie: 1,
      nacdate: 1,
      nationality: 1,
      email: 1,
      phone: 1,
      address: 1,
      nua: 1,
      account: 1,
      gender: 1,
      biometricId: 1,
      photo: 1,

      senador_data: {
        $cond: {
          if: { $eq: [{ $type: "$datos_senador" }, "object"] },
          then: {
            departamento: "$datos_senador.departamento",
            sigla: "$datos_senador.sigla",
            tipo: {
              $switch: {
                branches: [
                  {
                    case: { $and: [{ $gt: ["$titular_relacionado.ci", null] }, { $ne: ["$titular_relacionado.ci", ""] }] },
                    then: "SUPLENTE",
                  },
                  {
                    case: { $and: [{ $gt: ["$datos_senador.suplente", null] }, { $ne: ["$datos_senador.suplente", ""] }] },
                    then: "TITULAR",
                  },
                  {
                    case: { $gt: ["$datos_senador.tipo", null] },
                    then: "$datos_senador.tipo",
                  },
                  {
                    case: { $gt: ["$datos_senador.type", null] },
                    then: "$datos_senador.type",
                  },
                ],
                default: "TITULAR",
              },
            },
            su_suplente_ci: "$datos_senador.suplente",
            su_titular_ci: { $ifNull: ["$datos_senador.titular", "$titular_relacionado.ci"] },
            gestion: "$datos_senador.gestion",
            active: "$datos_senador.active",
          },
          else: "$$REMOVE",
        },
      },

      dependencia: {
        $switch: {
          branches: [
            {
              case: { $eq: [{ $type: "$funcionario_eventual" }, "object"] },
              then: "$funcionario_eventual.unit_data.unit",
            },
            {
              case: {
                $and: [
                  { $eq: [{ $type: "$funcionario_permanente" }, "object"] },
                  { $eq: ["$funcionario_permanente.active", true] },
                  { $eq: [{ $type: "$funcionario_permanente.disconnection" }, "missing"] },
                ],
              },
              then: "$funcionario_permanente.item_data.unit",
            },
          ],
          default: "$$REMOVE",
        },
      },

      cargo: {
        $switch: {
          branches: [
            {
              case: { $eq: [{ $type: "$funcionario_eventual" }, "object"] },
              then: "$funcionario_eventual.scale_data.position",
            },
            {
              case: {
                $and: [
                  { $eq: [{ $type: "$funcionario_permanente" }, "object"] },
                  { $eq: ["$funcionario_permanente.active", true] },
                  { $eq: [{ $type: "$funcionario_permanente.disconnection" }, "missing"] },
                ],
              },
              then: "$funcionario_permanente.item_data.denomination",
            },
          ],
          default: "$$REMOVE",
        },
      },

      funcionario_permanente: {
        $cond: {
          if: {
            $and: [
              { $eq: [{ $type: "$funcionario_permanente" }, "object"] },
              { $eq: ["$funcionario_permanente.active", true] },
              { $eq: [{ $type: "$funcionario_permanente.disconnection" }, "missing"] },
            ],
          },
          then: {
            item_data: "$funcionario_permanente.item_data",
            active: "$funcionario_permanente.active",
            disconnection: "$funcionario_permanente.disconnection",
          },
          else: "$$REMOVE",
        },
      },

      funcionario_eventual: {
        $cond: {
          if: { $eq: [{ $type: "$funcionario_eventual" }, "object"] },
          then: {
            unit_data: "$funcionario_eventual.unit_data",
            scale_data: "$funcionario_eventual.scale_data",
            endDate: "$funcionario_eventual.endDate",
          },
          else: "$$REMOVE",
        },
      },

      tipo_funcionario: {
        $switch: {
          branches: [
            {
              case: { $and: [{ $eq: [{ $type: "$datos_senador" }, "object"] }, { $gt: ["$titular_relacionado.ci", null] }] },
              then: "SENADOR_SUPLENTE",
            },
            {
              case: { $eq: [{ $type: "$datos_senador" }, "object"] },
              then: "SENADOR_TITULAR",
            },
            {
              case: { $eq: [{ $type: "$funcionario_eventual" }, "object"] },
              then: "FUNCIONARIO_EVENTUAL",
            },
            {
              case: {
                $and: [
                  { $eq: [{ $type: "$funcionario_permanente" }, "object"] },
                  { $eq: ["$funcionario_permanente.active", true] },
                  { $eq: [{ $type: "$funcionario_permanente.disconnection" }, "missing"] },
                ],
              },
              then: "FUNCIONARIO_PERMANENTE",
            },
          ],
          default: null,
        },
      },
    },
  },
  // 8. Filtrar finales válidos
  {
    $match: {
      $and: [
        {
          $or: [{ funcionario_permanente: { $exists: true } }, { funcionario_eventual: { $exists: true } }],
        },
        {
          $nor: [
            {
              $and: [
                {
                  $or: [{ tipo_funcionario: "SENADOR_TITULAR" }, { tipo_funcionario: "SENADOR_SUPLENTE" }],
                },
                { senador_data: { $exists: false } },
              ],
            },
          ],
        },
      ],
    },
  },
  // 9. Limpiar campos temporales y Salida
  {
    $project: { funcionario_permanente: 0, funcionario_eventual: 0 },
  },
  {
    $out: "view_people_pasajes",
  },
];

db.getCollection("peoples").aggregate(pipeline);

print("Colección creada 'view_people_pasajes' con éxito.");
